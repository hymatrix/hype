# Hype Openclaw WebUI

Local Web UI for `hype openclaw` commands: `spawn`, `conf-tg`, `pair-tg`, `chat`.

## Features

- Local-only API server (`127.0.0.1:7788`)
- Executes `hype openclaw <subcommand> --json`
- Displays structured JSON + raw stdout/stderr
- Sensitive fields are masked in displayed command preview
- In-memory secrets only (no localStorage persistence)

## Binary Resolution

The backend resolves `hype` binary in this order:

1. `<repo>/build/hype`
2. `hype` from `$PATH`

## Run (Production-style single entry)

```bash
cd /Users/webbergao/work/src/HymxWorkspace/hype
./frontend/scripts/start.sh
```

Then open: `http://127.0.0.1:7788`

## Run (Development)

```bash
cd /Users/webbergao/work/src/HymxWorkspace/hype
./frontend/scripts/dev.sh
```

- API: `http://127.0.0.1:7788`
- Vite: `http://127.0.0.1:5173`

## Backend Env

- `OPENCLAW_WEBUI_LISTEN` (default `127.0.0.1:7788`)
- `OPENCLAW_WEBUI_TIMEOUT_MS` (default `90000`)
- `HYPE_PRIVATE_KEY` or `PRV_KEY` may be used if UI privateKey is empty
