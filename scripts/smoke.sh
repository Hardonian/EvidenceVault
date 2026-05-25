#!/usr/bin/env bash
set -euo pipefail
go build -o /tmp/ev ./cmd/server
pkill -f '/tmp/ev' >/dev/null 2>&1 || true
CRON_SECRET=secret ADDR=:18080 PERSISTENCE_MODE=memory /tmp/ev >/tmp/ev.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT
sleep 1
curl -fsS http://127.0.0.1:18080/healthz >/dev/null
kill $PID; wait $PID || true
DATA_DIR=$(mktemp -d)
CRON_SECRET=secret ADDR=:18081 PERSISTENCE_MODE=file DATA_DIR=$DATA_DIR /tmp/ev >/tmp/ev2.log 2>&1 &
PID2=$!
trap 'kill $PID2 2>/dev/null || true' EXIT
sleep 1
curl -fsS http://127.0.0.1:18081/readyz >/dev/null
kill $PID2; wait $PID2 || true
