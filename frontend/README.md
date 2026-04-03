# Hype WebUI

Local Web UI for `hype openclaw` commands (`spawn`, `conf-tg`, `pair-tg`, `chat`) and `hype vmdocker` commands (`get`, `init`).

## Features

- Embedded in the `hype` binary via `hype ui`
- Local-only API server (`127.0.0.1:7788`)
- Executes the current `hype` binary for `openclaw` and `vmdocker` subcommands
- Top-level `.env` import stays in memory only and supports both file picker import and explicit path loading
- `View Env` shows all imported env variables in a modal
- Import prefills matching fields including `openclaw spawn.scheduler` from `VMDOCKER_SCHEDULER`
- Spawn form supports `provider` and `runtimeBackend`
- `defaultModel` / `defaultProvider` are derived automatically from the effective `model` / `provider`
- Spawn form can also auto-run `conf-tg` after spawn when `botToken` is filled
- Displays structured JSON + raw stdout/stderr
- Sensitive fields are masked in displayed command preview
- In-memory secrets only (no localStorage persistence)
- Relative paths are resolved from the current working directory used to start `hype ui`

## Run (Installed binary)

```bash
hype ui
```

Then open: `http://127.0.0.1:7788`

If you launch from a `build/` directory, relative paths such as `./.env` and `./vmdocker` resolve from that `build/` directory.

## Run (Repo local)

```bash
cd /Users/webbergao/work/src/HymxWorkspace/hype
go run ./cmd/hype ui
```

Then open: `http://127.0.0.1:7788`

## Run (Development)

```bash
cd /Users/webbergao/work/src/HymxWorkspace/hype
./frontend/scripts/dev.sh
```

- API: `http://127.0.0.1:7788`
- Vite: `http://127.0.0.1:5173`
- Backend: `go run ./cmd/hype ui --listen 127.0.0.1:7788`

## Backend Env

- `OPENCLAW_WEBUI_LISTEN` (default `127.0.0.1:7788`)
- `OPENCLAW_WEBUI_TIMEOUT_MS` (default `90000`)
- `HYPE_PRIVATE_KEY` or `PRV_KEY` may be used if UI privateKey is empty
