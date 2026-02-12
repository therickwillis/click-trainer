#!/bin/bash
set -e

echo "==> Installing Go tools..."
go install github.com/air-verse/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

echo "==> Downloading Go dependencies..."
cd /workspace
go mod download

echo "==> Running database migrations..."
# Wait for DB to accept connections, then run the app briefly to trigger auto-migrate
until pg_isready -h db -U clicktrainer; do sleep 1; done

echo "==> Dev environment ready!"
echo "    Run 'air' to start the server with hot-reload"
echo "    Run 'go test ./...' to run all tests"
