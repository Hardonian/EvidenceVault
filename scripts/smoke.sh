#!/usr/bin/env bash
set -euo pipefail
go build -o /tmp/ev ./cmd/server
CRON_SECRET=secret ADDR=:18080 /tmp/ev >/tmp/ev.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT
sleep 1
curl -fsS http://127.0.0.1:18080/healthz >/dev/null
curl -fsS http://127.0.0.1:18080/readyz >/dev/null
curl -fsS http://127.0.0.1:18080/version >/dev/null
kill $PID
wait $PID || true
