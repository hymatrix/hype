# hype
**hype** is the official project scaffolding and management CLI tool for the entire **hymx** Node.

## Build
- Use `make`:
  - `make build`
  - Output binary: `build/hype`
- Or use `go` directly:
  - `go build -o build/hype ./cmd/hype`

## Install
- From local source:
  - `go install ./cmd/hype`
- From GitHub with version:
  - `go install github.com/hymatrix/hype/cmd/hype@v0.0.2`
  or
  - `go install github.com/hymatrix/hype/cmd/hype@latest`
- Ensure `$(go env GOPATH)/bin` (or `GOBIN`) is in your `PATH`.

## Usage
- Basic:
  - `hype new -m <goModule> [-o <outDir>]`
  - `hype vmm --name <vmm> --format <format>`
  - `hype mount --name <vmm>`
  - `hype module --name <module> [-u <nodeURL>] [-k <privateKey>]`
  - `hype run [--mode <mode>]`
  - `hype db-import --redis-url <redisURL> --file <jsonl> [--force]`
  - `hype db-export --redis-url <redisURL> --pid <pid> --out <out> [--progress-every <n>]`
  - `hype openclaw spawn -m <moduleId> -s <scheduler> --model <model> [--timeout-ms <ms>] --api-key <key> --gateway-token <token> -k <privateKey> [-u <nodeURL>]`
  - `hype openclaw conf-tg -p <pid> --bot-token <token> [--default-account <account>] [--dm-policy <policy>] [--allow-from <value>] -k <privateKey> [-u <nodeURL>]`
  - `hype openclaw pair-tg -p <pid> -c <pairCode> [--channel telegram] [--dm-policy pairing] -k <privateKey> [-u <nodeURL>]`
  - `hype openclaw chat -p <pid> -c <command> -k <privateKey> [-u <nodeURL>]`
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
  - `--private-key`, `-k`: Private key; also supports env fallback `HYPE_PRIVATE_KEY` and `PRV_KEY`.
  - `--json`: Print JSON output.
- Notes:
  - `spawn` requires all of: `--module-id`, `--scheduler`, `--model`, `--api-key`, `--gateway-token`.
  - `spawn` sets `Container-Env-OPENCLAW_TIMEOUT_MS`; default is `180000`.
  - `conf-tg` requires `--bot-token`; `--default-account` defaults to `main`; `--dm-policy` defaults to `pairing`.

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
