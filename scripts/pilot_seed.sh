#!/bin/sh
set -eu
: "${CRON_SECRET:=pilot-secret}"
: "${BASE_URL:=http://127.0.0.1:8080}"
export APP_ENV=${APP_ENV:-development}
export DEMO_SEED=true
export PERSISTENCE_MODE=${PERSISTENCE_MODE:-file}
export DATA_DIR=${DATA_DIR:-./data}
printf 'Starting with demo seed enabled (APP_ENV=%s)\n' "$APP_ENV"
go run ./cmd/server >/tmp/evidencevault-pilot-seed.log 2>&1 &
PID=$!
trap 'kill "$PID" 2>/dev/null || true' EXIT INT TERM
sleep 2
curl -fsS "$BASE_URL/healthz" >/dev/null
printf 'Demo seed ready.\n'
