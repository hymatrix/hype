# hype
**hype** is the official project scaffolding and management CLI tool for the entire **hymx** Node.

Claude command usage guide:

- [`tools/claude/README.md`](/Users/webbergao/work/src/HymxWorkspace/hype/tools/claude/README.md)

## Build
- Use `make`:
  - `make build`
  - Output binary: `build/hype`
- Or use `go` directly:
  - Build frontend assets first:
    - `npm --prefix ./frontend ci`
    - `npm --prefix ./frontend run build`
  - `go build -o build/hype ./cmd/hype`

## Install
- From local source:
  - Build frontend assets first:
    - `npm --prefix ./frontend ci`
    - `npm --prefix ./frontend run build`
  - `go install ./cmd/hype`
  - or use `make install`
- From GitHub with version:
  - `go install github.com/hymatrix/hype/cmd/hype@v0.0.8`
  or
  - `go install github.com/hymatrix/hype/cmd/hype@latest`
- From npm:
  - `npm install -g @hymx/hype`
  - `npx @hymx/hype --version`
- Ensure `$(go env GOPATH)/bin` (or `GOBIN`) is in your `PATH`.
- The npm package downloads the matching GitHub Release binary during install.
- npm distribution currently supports macOS/Linux on `x64` and `arm64`.

## Release Flow
- Push a `v*` tag to trigger GoReleaser and upload GitHub Release assets.
- Publishing a GitHub Release triggers npm publish for `@hymx/hype`.
- The npm publish job verifies that `checksums.txt` and all supported tarballs are reachable before publishing.
- Configure `NPM_TOKEN` in GitHub Actions secrets for npm publishing.

## Usage
- Basic:
  - `hype new -m <goModule> [-o <outDir>]`
  - `hype vmm --name <vmm> --format <format>`
  - `hype mount --name <vmm>`
  - `hype module --name <module> [-u <nodeURL>] [-k <privateKey>]`
  - `hype run [--mode <mode>]`
  - `hype vmdocker get [--dir ./vmdockerv2] [--ref main]`
  - `hype vmdocker init [--dir ./vmdockerv2] --env-file <path>`
  - `hype vmdocker profile init --dir <agent-dir> --from <base-image>`
  - `hype vmdocker module build --dir ./vmdockerv2 --profile <profile.toml> --agent-bin <vmdocker-agent> --private-key <key>`
  - `hype vmdocker spawn --module-id <id> --scheduler <address> --runtime-type <type> [--runtime-backend docker|sandbox] [--env KEY=VALUE]`
  - `hype vmdocker export --pid <pid> --private-key <key>`
  - `hype db-import --redis-url <redisURL> --file <jsonl> [--force]`
  - `hype db-export --redis-url <redisURL> --pid <pid> --out <out> [--progress-every <n>]`
  - `hype openclaw spawn -m <moduleId> -s <scheduler> [--model <model>] [--provider <provider>] [--api-key <key>] --gateway-token <token> [--runtime-backend <docker|sandbox>] [--bot-token <token> --default-account <account> --dm-policy <policy> --allow-from <value>] -k <privateKey> [-u <nodeURL>]`
  - `hype openclaw conf-tg -p <pid> --bot-token <token> [--default-account <account>] [--dm-policy <policy>] [--allow-from <value>] -k <privateKey> [-u <nodeURL>]`
  - `hype openclaw pair-tg -p <pid> -c <pairCode> [--channel telegram] [--dm-policy pairing] -k <privateKey> [-u <nodeURL>]`
  - `hype openclaw chat -p <pid> -c <command> -k <privateKey> [-u <nodeURL>]`
  - `hype claude spawn -m <moduleId> -s <scheduler> --api-key <key> [--base-url <url>] [--model <model>] [--code-flags <flags>] [--runtime-backend <docker|sandbox>] -k <privateKey> [-u <nodeURL>]`
  - `hype claude chat -p <pid> -c <command> -k <privateKey> [-u <nodeURL>]`
  - `hype claude exec --pid <pid> -p <prompt> -k <privateKey> [-u <nodeURL>]`
  - `hype claude -p <prompt> --pid <pid> -k <privateKey> [-u <nodeURL>]`
- Version:
  - `hype -v` or `hype --version`
- REPL (interactive mode):
  - `hype` (no args) enters REPL by default
  - `hype repl` enters REPL explicitly
- If using local build without install:
  - `./build/hype ...`
- Specify output directory:
  - `hype new -m github.com/<user>/<pkg> -o ./_sandbox`
- The command generates a scaffolded Go project under the specified base directory; package name is derived from the output directory name, and the Go module path is set via `-m`.

### REPL Mode
- Start:
  - `hype` or `hype repl`
  - with private key preset for Openclaw commands:
    - `HYPE_PRIVATE_KEY=0x... ./hype`
- Help & exit:
  - `help` / `?`: show `hype --help`
  - `exit` / `quit` or Ctrl-D: exit REPL
- Run existing commands:
  - Type subcommands directly (no `hype` prefix needed), e.g. `version`, `new -m ...`, `db-export ...`
- Interactive required flags:
  - If you omit required flags in REPL, hype will prompt you to enter them.
  - If multiple required flags are missing, hype will prompt multiple times (one by one).
- Shell escape:
  - Prefix a line with `!` to execute it via `bash`, e.g. `!ls`, `!echo hello`

### Command: new
- Description: Create a new Golang project scaffold for hymx Node.
- Flags:
  - `--out`, `-o`: Output base directory. Default: `.`.
  - `--module`, `-m`: Go module name (e.g., `github.com/hymatrix/hype-example`). Required.
- Flow:
  - Run: `hype new -m github.com/<user>/<pkg> -o ./_sandbox`
  - The tool initializes `go.mod` using the provided module and runs `go mod tidy` inside the generated project.

### Command: vmm
- Description: Manage or scaffold a VM module, and auto-mount it into `cmd/main.go`.
- Flags:
  - `--name`, `-n`: Name of the vmm
  - `--format`, `-f`: Module format of the vmm. e.g. `hymx.token.foo.0.0.1`
- Notes:
  - Automatically inserts imports (`<vmm>`, `<vmm>Schema`) and adds `s.Mount(<vmm>Schema.ModuleFormat, <vmm>.Spawn)` into `cmd/main.go`.
  - Run inside the generated project root (the directory containing `cmd/main.go`).

### Command: mount
- Description: Mount an existing VM module into `cmd/main.go`.
- Flags:
  - `--name`, `-n`: Name of the vmm
- Notes:
  - Reads the module format from `<projectDir>/<vmm>/schema/schema.go`.
  - Inserts imports and the `s.Mount(...)` call under the mount hint comments in `cmd/main.go`.

### Command: module
- Description: Generate and mount a module by name, reading module format from schema.
- Flags:
  - `--name`, `-n`: Name of the module
  - `--node-url`, `-u`: Node URL. Default: `http://127.0.0.1:8080`
  - `--private-key`, `-k`: Private key
- Notes:
  - Reads `ModuleFormat` from `<projectDir>/<name>/schema/schema.go`.
  - Passes `node-url` and `private-key` to the generation pipeline.
  - Mounts the module into `cmd/main.go` using the same insertion logic as `vmm`.
  - When saving a module via SDK, generated `mod-<itemId>.json` will be placed under `cmd/mod/`.

### Command: run
- Description: Run the generated project.
- Flags:
  - `--mode`, `-m`: Start mode (`normal` or `rebuild`). Default: `normal`.
- Flow:
  - Executes: `cd cmd && go run ./ [--mode <mode>]`
  - From the generated project root, runs the `cmd/main.go` entrypoint.

### Command: vmdocker
- Description: Fetch and use the VMDocker V2 profile workflow from the CLI.
- Subcommands:
  - `get`: clone a `vmdockerv2` ref and build `build/hymx-node`
  - `init`: start Redis, start the built node in daemon mode, wait for health, then run `go run ./examples init`
  - `profile init`: create a testagent-derived `profile.toml` scaffold
  - `module build`: delegate module creation to `go run ./cmd/module` inside the V2 checkout
  - `spawn`: spawn any VMDocker V2 module with generic runtime tags
  - `export`: export a running process into a reusable module ID

#### `vmdocker get`
- Flags:
  - `--ref`: Git branch, tag, or commit. Default: `main`.
  - `--dir`: target clone directory. Default: `./vmdockerv2`.
- Behavior:
  - Clones `https://github.com/cryptowizard0/vmdockerv2.git`
  - Reuses an existing checkout only if it is already a `vmdockerv2` repo
  - Fetches the requested ref, refuses to switch when tracked files are dirty, and checks out the target commit detached
  - Builds `./build/hymx-node`
- Example:
  - `hype vmdocker get`
  - `hype vmdocker get --ref feature/profile --dir ./_sandbox/vmdockerv2`

#### `vmdocker init`
- Flags:
  - `--dir`: VMDocker checkout directory. Default: `./vmdockerv2`.
  - `--env-file`: `.env` file used for `examples init`. Required.
- Behavior:
  - Requires `<dir>/build/hymx-node` to exist
  - Starts Redis via Docker container `hype-vmdocker-redis` on port `6379`
  - Starts the node with `./build/hymx-node start --config ./cmd/config.yaml`
  - Waits for `http://127.0.0.1:8080/info`
  - Parses the provided `.env` file and injects it into `go run ./examples init`
- `.env` requirements:
  - Must include `VMDOCKER_PRIVATE_KEY`
  - `VMDOCKER_URL=http://127.0.0.1:8080` is injected automatically
- Example:
  - `hype vmdocker init --env-file ./local.env`

#### `vmdocker profile init`
- Flags:
  - `--dir`: target agent profile directory. Required.
  - `--from`: full base image name used as `FROM` in `profile.toml`. Required.
- Behavior:
  - Creates a minimal profile scaffold derived from `vmdockerv2/testagent`.
  - Writes `profile.toml`, `bin/.keep`, `skills/soul.md`, and `persona/style.md`.
  - Refuses to write into a non-empty directory.
- Example:
  - `hype vmdocker profile init --dir ./agent --from docker/sandbox-templates:claude-code`

#### `vmdocker module build`
- Flags:
  - `--dir`: VMDocker V2 checkout. Default: `./vmdockerv2`.
  - `--profile`: path to `profile.toml`. Required.
  - `--agent-bin`: path to the `vmdocker-agent` binary. May also come from `VMDOCKER_AGENT_BIN`.
  - `--node-url`: node URL. Precedence: flag, `VMDOCKER_URL`, checkout `.env`, then `http://127.0.0.1:8080`.
  - `--private-key`: module signing private key. Precedence: flag, `VMDOCKER_PRIVATE_KEY`, `HYPE_PRIVATE_KEY`, `PRV_KEY`, then checkout `.env`.
- Behavior:
  - Runs `go run ./cmd/module --profile <profile> --agent-bin <agent>` in the V2 checkout.
  - Streams stdout and stderr from the delegated command.
  - Does not build or download the agent binary.
  - Does not parse command output or write module IDs into `.env`.
- Example:
  - `hype vmdocker module build --dir ./vmdockerv2 --profile ./agent/profile.toml --agent-bin ./bin/vmdocker-agent --private-key 0x...`

#### `vmdocker spawn`
- Flags:
  - `--module-id`, `-m`: module ID. Precedence: flag, `VMDOCKER_MODULE_ID`.
  - `--scheduler`, `-s`: scheduler address. Precedence: flag, `VMDOCKER_SCHEDULER`.
  - `--runtime-type`: runtime type. Precedence: flag, `RUNTIME_TYPE`.
  - `--runtime-backend`: `docker` or `sandbox`. Precedence: flag, `RUNTIME_BACKEND`.
  - `--env`: repeatable `KEY=VALUE` container environment assignment.
  - `--node-url`, `-u`: node URL. Precedence: flag, `VMDOCKER_URL`, then `http://127.0.0.1:8080`.
  - `--private-key`, `-k`: signer private key. Precedence: flag, `VMDOCKER_PRIVATE_KEY`, `HYPE_PRIVATE_KEY`, `PRV_KEY`.
  - `--json`: print JSON output.
- Tag mapping:
  - `--runtime-type claude` -> `Container-Env-RUNTIME_TYPE=claude`
  - `--env TOKEN=a=b` -> `Container-Env-TOKEN=a=b`
  - `--runtime-backend docker` -> `Runtime-Backend=docker`
  - `RUNTIME_TYPE` is reserved for `--runtime-type`; duplicate `--env` keys are rejected.
- Behavior:
  - Prints the spawned process ID only; it never writes the pid into `.env`.
- Example:
  - `hype vmdocker spawn --module-id <id> --scheduler <address> --runtime-type claude --runtime-backend sandbox --env TOKEN=a=b --private-key 0x...`

#### `vmdocker export`
- Flags:
  - `--pid`, `-p`: process ID. Precedence: flag, `VMDOCKER_EXPORT_PID`.
  - `--node-url`, `-u`: node URL. Precedence: flag, `VMDOCKER_URL`, then `http://127.0.0.1:8080`.
  - `--private-key`, `-k`: signer private key. Precedence: flag, `VMDOCKER_PRIVATE_KEY`, `HYPE_PRIVATE_KEY`, `PRV_KEY`.
  - `--json`: print JSON output.
- Behavior:
  - Sends the VMDocker `Action=Export` message to the running process.
  - Strictly decodes the node result and prints `export ok, module id: <id>` on success.
  - The returned module ID can be passed back to `hype vmdocker spawn --module-id <id>`.
  - It never writes the module ID into `.env`.
- Example:
  - `hype vmdocker export --pid <pid> --private-key 0x...`

> Note: the embedded Web UI's VMDocker Get action still emits the removed `--version` flag and is unsupported by this CLI-only V2 migration.

### Command: db-import
- Description: Import a JSONL data file into Redis, calling IDB.Commit for each item.
- Flags:
  - `--redis-url`, `-r`: Redis connection URL (e.g. `redis://@localhost:6379/0`). Required.
  - `--file`, `-f`: Path to JSONL file. Required.
  - `--force`, `-F`: If set, write even if msg.Id already exists; otherwise skip duplicates.
- JSONL format:
  - Each line is an object with `{pid, nonce, msg, assign}`.
- Flow:
  - Sorts items by `nonce`, validates start at 0 and continuous increments.
  - Checks existing messages via `GetMessage(msg.Id)`.
    - Without `--force`: skip if exists.
    - With `--force`: append regardless.
  - Calls `Commit(pid, nonce, msg, assign)` for each item.
- Example:
  - `./build/hype db-import --redis-url redis://@localhost:6379/0 --file ./data.jsonl`

### Command: db-export
- Description: Export a process from Redis into JSONL (supports .gz).
- Flags:
  - `--redis-url`, `-r`: Redis connection URL. Required.
  - `--pid`, `-p`: Process id. Required.
  - `--out`, `-o`: Output file path (e.g. `./data.jsonl` or `./data.jsonl.gz`). Required.
  - `--progress-every`: Print progress every N lines. Default: 1000.
- Example:
  - `./build/hype db-export --redis-url redis://@localhost:6379/0 --pid process-123 --out ./process-123.jsonl`
  - `./build/hype db-export --redis-url redis://@localhost:6379/0 --pid process-123 --out ./process-123.jsonl.gz`


### Command: openclaw
- Description: Execute Openclaw runtime workflows through hymx SDK.
- Subcommands:
  - `spawn`: Create a new Openclaw process using module + scheduler + spawn tags.
  - `conf-tg`: Configure Telegram runtime settings (`ConfigureTelegram`).
  - `pair-tg`: Approve Telegram pairing (`ApproveTelegramPairing`).
  - `chat`: Send a chat message (`Chat`).
- Shared Flags:
  - `--node-url`, `-u`: Node URL. Default: `http://127.0.0.1:8080`.
  - `--private-key`, `-k`: Private key; fallback order is `--private-key` > `HYPE_PRIVATE_KEY` > `PRV_KEY`.
  - `--json`: Print JSON output.
- Notes:
  - `spawn` requires `--module-id`, `--scheduler`, `--gateway-token`.
  - `spawn` tag layout is aligned with `vmdocker/examples/openclaw.go`: optional `provider`, `model`, `apiKey`, `Container-Env-OPENCLAW_GATEWAY_TOKEN`, optional `Runtime-Backend`, plus derived `Container-Env-OPENCLAW_DEFAULT_MODEL` and `Container-Env-OPENCLAW_DEFAULT_PROVIDER`.
  - `hype` now normalizes `model` and `provider` before sending tags. `model=opencode-go/kimi-k2.5` with empty `provider` is treated the same as `model=kimi-k2.5 --provider opencode-go`.
  - After normalization, `OPENCLAW_DEFAULT_MODEL` and `OPENCLAW_DEFAULT_PROVIDER` always mirror the effective `model` and `provider`.
  - If `--api-key` is provided and `--model` has no provider prefix like `zen/...`, then `--provider` is required.
  - If both `--provider` and a provider-prefixed `--model` are set, they must agree.
  - `spawn` may set `Runtime-Backend` at spawn time. If omitted, `vmdocker` chooses by OS: macOS prefers `sandbox`, Linux prefers `docker`.
  - Runtime workspace root is no longer configurable from `hype`; `vmdocker` always uses its default workspace layout.
  - If `spawn` also receives `--bot-token`, it will immediately send `ConfigureTelegram` to the newly spawned pid.
  - In this auto-`conf-tg` flow, `--default-account` defaults to `main`, `--dm-policy` defaults to `open`, and `--allow-from` defaults to `*`.
  - `conf-tg` requires `--bot-token`; `--default-account` defaults to `main`; `--dm-policy` defaults to `pairing`; `--allow-from` defaults to `*`.
  - Open DM mode still requires `allow-from` to include `*`; the default already satisfies that.

Example spawn with explicit sandbox backend:

```bash
./build/hype openclaw spawn \
  --module-id <moduleId> \
  --scheduler <scheduler> \
  --model kimi-k2.5 \
  --provider opencode-go \
  --api-key <providerApiKey> \
  --gateway-token openclaw-test-token \
  --runtime-backend sandbox \
  --bot-token <telegramBotToken> \
  --default-account main \
  --dm-policy open \
  --allow-from '*' \
  --private-key <privateKey>
```

### Command: claude
- Description: Execute Claude runtime workflows through hymx SDK.
- Detailed usage:
  - [`tools/claude/README.md`](/Users/webbergao/work/src/HymxWorkspace/hype/tools/claude/README.md)
- Subcommands:
  - `spawn`: Create a new Claude process using module + scheduler + Claude env tags.
  - `chat`: Send a chat message (`Chat`).
  - `exec`: Send a generic prompt (`Execute`).
- Shared Flags:
  - `--node-url`, `-u`: Node URL. Default: `http://127.0.0.1:8080`.
  - `--private-key`, `-k`: Private key; fallback order is `--private-key` > `HYPE_PRIVATE_KEY` > `PRV_KEY`.
  - `--json`: Print JSON output.
- Notes:
  - `spawn` requires `--module-id`, `--scheduler`, `--api-key`.
  - `spawn` env/tag layout is aligned with the Claude runtime in `vmdocker_agent`: `Container-Env-RUNTIME_TYPE=claude`, `Container-Env-ANTHROPIC_API_KEY`, optional `Container-Env-ANTHROPIC_BASE_URL`, optional `Container-Env-ANTHROPIC_MODEL`, optional `Container-Env-CLAUDE_CODE_FLAGS`, and optional `Runtime-Backend`.
  - `spawn` reads these env vars when flags are omitted: `VMDOCKER_MODULE_ID`, `VMDOCKER_SCHEDULER`, `RUNTIME_BACKEND`, `ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, `CLAUDE_CODE_FLAGS`.
  - `chat` sends `Action=Chat`.
  - `exec` sends `Action=Execute`.
  - `hype claude -p "..." --pid <pid>` is a shortcut for `hype claude exec --pid <pid> -p "..."`.

Example Claude spawn:

```bash
./build/hype claude spawn \
  --module-id <moduleId> \
  --scheduler <scheduler> \
  --api-key <anthropicApiKey> \
  --base-url <anthropicBaseURL> \
  --model <anthropicModel> \
  --runtime-backend sandbox \
  --private-key <privateKey>
```

Example Claude exec:

```bash
./build/hype claude exec \
  --pid <pid> \
  --prompt "你好，请介绍一下自己" \
  --private-key <privateKey>
```

## Hype Web UI
- Description: Local Web UI embedded in the `hype` binary for `hype openclaw`, `hype claude`, and `hype vmdocker` commands.
- Claude usage guide:
  - [`tools/claude/README.md`](/Users/webbergao/work/src/HymxWorkspace/hype/tools/claude/README.md)
- Features:
  - Local-only API server (`127.0.0.1:7788`).
  - Start with `hype ui`.
  - Executes the current `hype` binary for `openclaw`, `claude`, and `vmdocker` subcommands.
  - Displays structured JSON and raw stdout/stderr.
  - Masks sensitive fields in command preview.
  - Keeps secrets in memory only (no localStorage persistence).
  - Imports a local `.env` file in memory by file picker or explicit path, prefills matching Openclaw fields, and uses it for `vmdocker init`.
  - `View Env` opens a modal showing every imported env entry.
  - `.env` import prefills `openclaw spawn` fields including `moduleId`, `scheduler` (`VMDOCKER_SCHEDULER`), `model`, `provider`, `apiKey`, and Telegram settings.
  - UI-relative paths are resolved from the current working directory used to start `hype ui`.
- Run (single entry):
  - `hype ui`
  - Open: `http://127.0.0.1:7788`
- Relative path note:
  - If you start with `cd build && ./hype ui`, paths such as `./.env` and `./vmdocker` resolve under `build/`.
  - If you start with `cd /path/to/hype && ./build/hype ui`, the same paths resolve under the repo root.
- Run from a repo checkout:
  - build frontend assets first, then:
  - `go run ./cmd/hype ui`
- Run (development):
  - `./frontend/scripts/dev.sh`
  - API: `http://127.0.0.1:7788`
  - Vite: `http://127.0.0.1:5173`
- Backend environment variables:
  - `OPENCLAW_WEBUI_LISTEN` (default `127.0.0.1:7788`)
  - `OPENCLAW_WEBUI_TIMEOUT_MS` (default `90000`)
  - `HYPE_PRIVATE_KEY` or `PRV_KEY` may be used if UI privateKey is empty
- More details: `frontend/README.md`

### Generated Structure
- Base path: `<out>/<pkg>/`
- Contents:
  - `cmd/`
    - `main.go`
    - `flags.go`
    - `const.go`
    - `cmds.go`
    - `cfgchainkit.go`
    - `cfgnode.go`
    - `cfgpay.go`
    - `config.yaml`
    - `config_chainkit.yaml`
    - `config_payment.yaml`
    - `config_test_network.yaml`
    - `mod/*.json` (copied from templates, `.tmpl` suffix removed)
    - `mod/mod-<itemId>.json` (when SDK-based module save is performed)
  - `<pkg>/<pkg>.go` (interface file)

### Build Generated Project
- From the generated project root:
  - `cd <out>/<pkg>`
  - `go build -o ./<pkg> ./cmd`

## Notes
- Dependencies used by the scaffold (e.g., `github.com/spf13/viper`, `github.com/urfave/cli/v2`, `github.com/hymatrix/hymx`) are fetched via `go mod tidy` during generation.

## TODO
- [x] Create project directories
- [x] Initialize Golang environment
- [x] Create Vmm & Mount
- [x] Generate Module
- [x] Run
- [ ] Upload & Download Module from arweave
- [ ] ENV support
- [ ] hymx sdk support
- [ ] Setup Redis environment
