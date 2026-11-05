#!/usr/bin/env bash
set -euo pipefail
cd /data/home/sb-agent
chmod +x sandbox-worker
killall -q sandbox-worker 2>/dev/null || true
: > worker.log
nohup env \
  SB_AGENT_TOKEN="${SB_AGENT_TOKEN:?}" \
  SANDBOX_WORKER_ADDR=:9090 \
  SANDBOX_WORKER_ID=instabox-sb-1 \
  SANDBOX_WORKSPACE=/data/home/sb-agent \
  CONTROL_PLANE_URL="${CONTROL_PLANE_URL:?}" \
  SANDBOX_PUBLIC_URL="${SANDBOX_PUBLIC_URL:?}" \
  ./sandbox-worker >> worker.log 2>&1 &
disown || true
AUTH=()
[[ -n "${SB_AGENT_TOKEN:-}" ]] && AUTH=(-H "Authorization: Bearer $SB_AGENT_TOKEN")
for _ in 1 2 3 4 5; do
  if curl -sf "${AUTH[@]}" http://127.0.0.1:9090/health >/dev/null; then
    exit 0
  fi
  sleep 1
done
echo "worker health check failed" >&2
tail -20 worker.log >&2 || true
exit 1
