#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

cd "$ROOT_DIR"
echo "[dev] starting api server on http://127.0.0.1:7788"
OPENCLAW_WEBUI_LISTEN=127.0.0.1:7788 go run ./frontend/server &
SERVER_PID=$!

cd "$FRONTEND_DIR"
echo "[dev] starting vite on http://127.0.0.1:5173"
npm run dev
