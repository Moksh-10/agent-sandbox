#!/usr/bin/env bash
# Dev helper: run one sandbox-worker on Instabox.
# Production: an orchestrator creates a VM per session and returns base_url from the API
# (see cmd/sandbox-orchestrator + SANDBOX_ALLOCATOR=orchestrator) — do not pin sandbox URL on control plane.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

IB="${INSTABOX_CLI:-npx --yes instabox@latest}"
NAME="${INSTABOX_MACHINE_NAME:-sb-agent}"
TOKEN="${SB_AGENT_TOKEN:?set SB_AGENT_TOKEN in .env}"

if ! $IB ls --json >/dev/null 2>&1; then
  echo "Instabox: not logged in. Run once:"
  echo "  npx instabox@latest login"
  echo "Or set INSTABOX_API_KEY in .env"
  exit 1
fi

echo "==> Resolve or create machine: $NAME"
MACHINE="$NAME"
if ! $IB ls --json | grep -q "\"name\": \"$NAME\""; then
  echo "Creating machine (free tier: ci template)..."
  $IB create --name "$NAME" --template ci
fi

echo "==> Build linux/amd64 sandbox-worker"
mkdir -p "$ROOT/bin"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$ROOT/bin/sandbox-worker-linux" ./cmd/sandbox-worker

echo "==> Sync project to Instabox"
$IB exec "$MACHINE" -- mkdir -p /data/home/sb-agent
$IB exec "$MACHINE" -- rm -rf /data/home/sb-agent/cmd /data/home/sb-agent/control /data/home/sb-agent/worker /data/home/sb-agent/types /data/home/sb-agent/config
$IB cp "$ROOT/bin/sandbox-worker-linux" "$MACHINE:/data/home/sb-agent/sandbox-worker"
$IB cp "$ROOT/go.mod" "$MACHINE:/data/home/sb-agent/go.mod"
[[ -f "$ROOT/go.sum" ]] && $IB cp "$ROOT/go.sum" "$MACHINE:/data/home/sb-agent/go.sum"
for dir in cmd control worker types config; do
  $IB cp -r "$ROOT/$dir" "$MACHINE:/data/home/sb-agent/"
done
$IB cp "$ROOT/scripts/instabox-start-worker.sh" "$MACHINE:/data/home/sb-agent/start-worker.sh"

echo "==> Tunnel control plane so the VM can register (cloudflared quick tunnel)"
if [[ -z "${CONTROL_PLANE_PUBLIC_URL:-}" ]]; then
  if ! command -v cloudflared >/dev/null; then
    echo "Install cloudflared, or set CONTROL_PLANE_PUBLIC_URL in .env to a URL reachable from Instabox."
    exit 1
  fi
  CF_LOG="$(mktemp)"
  CF_PID_FILE="$ROOT/.instabox-cloudflared.pid"
  if [[ -f "$CF_PID_FILE" ]]; then
    old_pid="$(cat "$CF_PID_FILE" 2>/dev/null || true)"
    kill "$old_pid" 2>/dev/null || true
  fi
  cloudflared tunnel --url "http://127.0.0.1:8080" >"$CF_LOG" 2>&1 &
  CF_PID=$!
  echo "$CF_PID" >"$CF_PID_FILE"
  for _ in $(seq 1 30); do
    CP_URL="$(grep -Eo 'https://[a-z0-9-]+\.trycloudflare\.com' "$CF_LOG" | head -1 || true)"
    [[ -n "$CP_URL" ]] && break
    sleep 1
  done
  if [[ -z "$CP_URL" ]]; then
    echo "cloudflared did not print a URL; see $CF_LOG"
    exit 1
  fi
  echo "CONTROL_PLANE_PUBLIC_URL=$CP_URL"
else
  CP_URL="$CONTROL_PLANE_PUBLIC_URL"
fi

echo "==> Publish port 9090 (HTTPS URL assigned by Instabox — changes per machine/port lifecycle)"
PUBLISH_OUT="$($IB port publish "$MACHINE" 9090 --public 2>&1 || true)"
echo "$PUBLISH_OUT"
SANDBOX_URL="$(echo "$PUBLISH_OUT" | grep -Eo 'https://[^[:space:]]+' | head -1 || true)"
if [[ -z "$SANDBOX_URL" ]]; then
  SANDBOX_URL="$($IB port ls "$MACHINE" --json 2>/dev/null | grep -Eo 'https://[^\"]+' | head -1 || true)"
fi
if [[ -z "$SANDBOX_URL" ]]; then
  echo "Could not parse published URL; run: instabox port ls $MACHINE"
  exit 1
fi

echo "==> Start worker inside Instabox (background) with dynamic SANDBOX_PUBLIC_URL=$SANDBOX_URL"
$IB exec "$MACHINE" -- chmod +x /data/home/sb-agent/start-worker.sh
if ! $IB exec "$MACHINE" -- env \
  SB_AGENT_TOKEN="$TOKEN" \
  CONTROL_PLANE_URL="$CP_URL" \
  SANDBOX_PUBLIC_URL="$SANDBOX_URL" \
  bash /data/home/sb-agent/start-worker.sh; then
  echo "Worker start failed; last log lines:"
  $IB exec "$MACHINE" -- tail -30 /data/home/sb-agent/worker.log || true
  exit 1
fi

CP_LOCAL="${CONTROL_PLANE_URL:-http://127.0.0.1:8080}"
echo "==> Register worker with control plane from laptop (pool mode)"
REGISTER_BODY=$(printf '{"id":"instabox-sb-1","base_url":"%s","capabilities":["read","write","shell","grep","exa_search","context7"]}' "$SANDBOX_URL")
curl -sf -X POST "$CP_LOCAL/internal/workers/register" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$REGISTER_BODY" >/dev/null

HB_PID_FILE="$ROOT/.instabox-heartbeat.pid"
if [[ -f "$HB_PID_FILE" ]]; then
  kill "$(cat "$HB_PID_FILE" 2>/dev/null)" 2>/dev/null || true
fi
(
  while true; do
    curl -sf -X POST "$CP_LOCAL/internal/workers/instabox-sb-1/heartbeat" \
      -H "Authorization: Bearer $TOKEN" >/dev/null || true
    sleep 10
  done
) &
echo $! >"$HB_PID_FILE"
echo "Local heartbeat PID $(cat "$HB_PID_FILE") (VM→tunnel register may still fail if cloudflared is down)"

echo ""
echo "Done."
echo "  1) Restart control-plane (loads .env): go run ./cmd/control-plane"
echo "  2) Stop local sandbox-worker if using Instabox pool worker only"
echo "     (orchestrator mode: set SANDBOX_ALLOCATOR=orchestrator — URL comes from POST /v1/sandboxes per session)"
echo "  3) Create session + run:"
echo "     curl -s -X POST http://127.0.0.1:8080/v1/sessions -H \"Authorization: Bearer $TOKEN\" -H 'Content-Type: application/json' -d '{\"message\":\"instabox\"}' | jq ."
echo ""
echo "Instabox worker logs: instabox exec $MACHINE -- tail -f /data/home/sb-agent/worker.log"
echo "Instabox sandbox URL: $SANDBOX_URL"
echo "Control plane tunnel: $CP_URL"
if [[ -n "${CF_PID:-}" ]]; then
  echo "cloudflared PID $CF_PID (kept running for worker heartbeats; stop: kill \$(cat .instabox-cloudflared.pid))"
fi
