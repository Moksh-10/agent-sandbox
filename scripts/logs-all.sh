#!/usr/bin/env bash
# Tail local + Instabox logs (control plane / worker should already be running).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
[[ -f .env ]] && set -a && source .env && set +a

IB="${INSTABOX_CLI:-npx --yes instabox@latest}"
NAME="${INSTABOX_MACHINE_NAME:-sb-agent}"

echo "=== Local terminals: control-plane :8080, sandbox-worker :9090 (if running) ==="
echo "=== Instabox worker log ($NAME) ==="
if $IB ls --json >/dev/null 2>&1; then
  $IB exec "$NAME" -- tail -n 80 /data/home/sb-agent/worker.log 2>/dev/null || echo "(no worker.log yet — run scripts/instabox-launch.sh)"
  echo ""
  echo "=== Instabox machine ops log ==="
  $IB logs "$NAME" 2>/dev/null | tail -n 40 || true
else
  echo "Instabox not logged in (npx instabox@latest login)"
fi
