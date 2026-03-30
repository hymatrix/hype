#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"

cd "$FRONTEND_DIR"
if [[ ! -d node_modules ]]; then
  echo "[start] installing frontend dependencies"
  npm install
fi

echo "[start] building frontend"
npm run build

cd "$ROOT_DIR"
echo "[start] serving on http://127.0.0.1:7788"
go run ./cmd/hype ui --listen 127.0.0.1:7788
