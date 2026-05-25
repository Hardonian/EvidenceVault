#!/usr/bin/env bash
set -euo pipefail
go test ./...
go build ./cmd/server
echo "smoke passed"
