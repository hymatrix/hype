# hype

> The official scaffolding and management CLI for the **hymx** Node.

**English** | [中文](README_zh.md)

`hype` gets you from an empty directory to a running module: scaffold a hymx project, then drive the VMDocker V2 workflow end to end — fetch, build, spawn, and export profile-based modules — from one binary or an interactive REPL.

- **New here?** Jump to [Quick Start](#quick-start) for a 5‑minute end-to-end run.
- **Looking for a command?** See the [Command Reference](#command-reference) cheat sheet.

---

## Table of Contents

1. [Overview](#overview)
2. [Install](#install)
3. [Quick Start](#quick-start)
4. [vmdocker Usage](#vmdocker-usage)
5. [Command Reference](#command-reference)
6. [Command Details](#command-details)
7. [Scaffolded Project Layout](#scaffolded-project-layout)
8. [Build from Source & Release](#build-from-source--release)
9. [Notes](#notes)

---

## Overview

**What it is.** `hype` (`github.com/hymatrix/hype`) is the command-line companion for the hymx Node. It scaffolds Go projects, manages VM modules, and drives the VMDocker V2 runtime end to end.

**What it does.**

- **Project scaffolding** — generate a hymx Node project and mount VM modules (`new`, `vmm`, `mount`, `module`, `run`, `get`).
- **VMDocker V2 workflow** — fetch, initialize, build, spawn, and export profile-based modules (`vmdocker …`).
- **Data migration** — move process data between Redis and JSONL (`db-import`, `db-export`).
- **Interactive** — a built-in REPL (`repl`, or just run `hype` with no args).

**Where it sits.** `hype` is the outermost orchestrator in the hymx stack:

```
hype (CLI / orchestrator)
  └── hymx            platform / node — exposes /vmm/* and mounts module formats
        └── vmdocker  the VM shell — builds images, spawns & schedules containers
              └── vmdocker-agent   in-image adapter (PID 1) that runs your runtime
```

**How you run it.** Pick whichever fits: one-shot subcommands (`hype <cmd>`), or the interactive REPL (`hype` with no args).

---

## Install

Choose one:

```bash
# npm (downloads the matching GitHub Release binary)
npm install -g @hymx/hype

# Go toolchain
go install github.com/hymatrix/hype/cmd/hype@latest   # or @v0.1.0

# From source
make build        # -> build/hype
```

Then make sure the binary is on your `PATH`:

- For `go install`, add `$(go env GOPATH)/bin` (or `GOBIN`) to `PATH`.
- npm distribution currently supports macOS/Linux on `x64` and `arm64`.

Verify:

```bash
hype --version    # or: hype -v
```

Throughout this guide, `hype …` and a local `./build/hype …` are interchangeable.

---

## Quick Start

This is the fastest path to a running process: the **VMDocker V2** workflow, from fetch to spawn.

### Prerequisites

- **Docker** and **Redis** available locally (used by `vmdocker init`).
- **Go 1.24+** (to build the fetched VMDocker node).
- A **signing private key** (`0x…`) for build / spawn / export.
- A **scheduler address** for spawn.
- A **`vmdocker-agent`** binary for module builds (`hype` does not build it for you).

### Steps

```bash
# 1. Fetch and build the VMDocker V2 checkout (clones to ./vmdockerv2)
hype vmdocker get

# 2. Start Redis + the local node, then run examples init.
#    ./local.env must contain: VMDOCKER_PRIVATE_KEY=0x...
hype vmdocker init --dir ./vmdockerv2 --env-file ./local.env

# 3. Scaffold an agent profile (target dir must be empty/absent)
hype vmdocker profile init --dir ./agent --from docker/sandbox-templates:claude-code

# 4. Build & sign a module from the profile
hype vmdocker module build \
  --dir ./vmdockerv2 \
  --profile ./agent/profile.toml \
  --agent-bin ./bin/vmdocker-agent \
  --private-key 0x...

# 5. Spawn a process from the module (prints a pid)
hype vmdocker spawn \
  --module-id <module-id> \
  --scheduler <scheduler-address> \
  --runtime-type claude \
  --runtime-backend sandbox \
  --private-key 0x...

# 6. (Optional) Export a running process into a reusable module ID
hype vmdocker export --pid <pid> --private-key 0x...
```

**Success looks like:** step 5 prints `spawn ok, pid: <pid>`, and step 6 prints `export ok, module id: <module-id>`. Pass the returned module ID straight back into `hype vmdocker spawn --module-id <id>`.

Prefer to explore interactively first? Run `hype` with no args to enter the REPL — it prompts you for any required flags you omit.

> Next: read [vmdocker Usage](#vmdocker-usage) for what each step does, or the deep-dive [VMDocker V2 CLI Workflow](docs/vmdocker-v2-cli-workflow.md).

---

## vmdocker Usage

The `vmdocker` command group drives the full VMDocker V2 lifecycle. `hype` prints IDs to stdout but **never writes module IDs or process IDs back into `.env`**.

### Core concepts

- **Module** — a signed, self-contained unit (`mod-<id>.json`) that carries the container image plus a `profile.toml`. Spawn loads the image from the module when it is not already present locally.
- **`profile.toml`** — a declarative recipe for a module: a `[dockerfile]` section (input to a standardized Dockerfile generator — you do not hand-write a Dockerfile) and a `[vmdocker]` section (the `public` export allowlist).
- **Spawn** — creates a process (pid) from a module. `--runtime-type` (e.g. `claude`) selects the in-image adapter's readiness behavior; `--runtime-backend` is `docker` or `sandbox`.
- **Export** — clones a running process's public state into a new, reusable module ID. There is no `respawn`; use `export`, then `spawn` with the returned ID.

### Minimal `profile.toml`

`hype vmdocker profile init` writes this scaffold (plus `bin/.keep`, `skills/soul.md`, `persona/style.md`):

```toml
[dockerfile]
# Full base image name, used verbatim as Dockerfile FROM.
FROM = "docker/sandbox-templates:claude-code"

# Directory of user executables, copied to /usr/local/bin. Required; may be empty.
bin = "bin"

# Optional startup command (Dockerfile CMD syntax); the adapter stays ENTRYPOINT.
# CMD = ["your-engine", "--serve"]

# Optional cross-distribution tool packages installed during the image build.
tools = []

# Optional Dockerfile RUN bodies (without the leading "RUN ").
RUN = []

[vmdocker]
# Export allowlist relative to HOME; everything else stays private.
public = ["~/skills/*", "~/persona/*", "~/.hermes/plugin/*"]
```

Only `[dockerfile].FROM` and `[dockerfile].bin` are required; `CMD` is optional.

### Workflow at a glance

| Step | Command | Purpose |
|------|---------|---------|
| Fetch | `vmdocker get` | Clone a `vmdockerv2` ref and build `build/hymx-node`. |
| Init | `vmdocker init` | Start Redis + node, wait for health, run examples init. |
| Profile | `vmdocker profile init` | Scaffold a `profile.toml` agent directory. |
| Build | `vmdocker module build` | Build & sign a module from the profile. |
| Spawn | `vmdocker spawn` | Spawn a process from a module (prints pid). |
| Export | `vmdocker export` | Export a running process into a new module ID. |
| Clean | `vmdocker clean` | Stop the local node and remove generated runtime files. |

Reset local state any time with:

```bash
hype vmdocker clean --dir ./vmdockerv2
```

For per-step behavior, input precedence, and tag mapping, see the full [VMDocker V2 CLI Workflow](docs/vmdocker-v2-cli-workflow.md) and [Command Details](#command-details) below.

---

## Command Reference

Run `hype <command> --help` for the authoritative, up-to-date flags. Required flags are **bold**.

### Scaffolding

| Command | What it does | Key flags |
|---------|--------------|-----------|
| `new` | Create a new hymx Go project scaffold | **`-m/--module`**, `-o/--out` (`.`) |
| `get` | Fetch a VMM package via go tooling and mount it | **`-p/--package`** |
| `vmm` | Scaffold a VM module and auto-mount into `cmd/main.go` | **`-n/--name`**, **`-f/--format`** |
| `mount` | Mount an existing VM module into `cmd/main.go` | **`-n/--name`** |
| `module` | Generate + mount a module by name | **`-n/--name`**, `-u/--node-url`, `-k/--private-key` |
| `run` | Run the generated project (`cd cmd && go run ./`) | `-m/--mode` (`normal`\|`rebuild`) |

### vmdocker

| Command | What it does | Key flags |
|---------|--------------|-----------|
| `vmdocker get` | Clone a `vmdockerv2` ref and build `hymx-node` | `--ref` (`main`), `--dir` (`./vmdockerv2`) |
| `vmdocker init` | Start Redis + node, run examples init | `--dir`, **`--env-file`** |
| `vmdocker profile init` | Scaffold a `profile.toml` agent directory | **`--dir`**, **`--from`** |
| `vmdocker module build` | Build & sign a module from a profile | **`--profile`**, `--agent-bin`, `--dir`, `--node-url`, `--private-key` |
| `vmdocker spawn` | Spawn a process from a module | **`-m/--module-id`**, **`-s/--scheduler`**, **`-k/--private-key`**, `--runtime-type`, `--runtime-backend`, `--env` |
| `vmdocker export` | Export a running process into a module ID | **`-p/--pid`**, **`-k/--private-key`** |
| `vmdocker clean` | Stop node + remove generated runtime files | `--dir` |

### Data

| Command | What it does | Key flags |
|---------|--------------|-----------|
| `db-import` | Import a JSONL file into Redis | **`-r/--redis-url`**, **`-f/--file`**, `-F/--force` |
| `db-export` | Export process data from Redis to JSONL | **`-r/--redis-url`**, **`-o/--out`**, `-p/--pid`, `--progress-every` |

### Interactive & Other

| Command | What it does | Key flags |
|---------|--------------|-----------|
| `repl` | Start interactive mode (also the default with no args) | — |
| `version` | Print version information | — |

---

## Command Details

### Common options & environment fallbacks

Several commands share these conventions:

- **`-u/--node-url`** — hymx node URL. Default `http://127.0.0.1:8080`. `vmdocker spawn`/`export` also read `VMDOCKER_URL`.
- **`-k/--private-key`** — signing key. Fallback order: `--private-key` → `HYPE_PRIVATE_KEY` → `PRV_KEY` → `VMDOCKER_PRIVATE_KEY`.
- **`--json`** — machine-readable output (`vmdocker spawn`/`export`).

| Command | Env fallbacks |
|---------|---------------|
| `vmdocker spawn` | `VMDOCKER_MODULE_ID`, `VMDOCKER_SCHEDULER`, `RUNTIME_TYPE`, `RUNTIME_BACKEND`, `VMDOCKER_URL`, `VMDOCKER_PRIVATE_KEY`/`HYPE_PRIVATE_KEY`/`PRV_KEY` |
| `vmdocker export` | `VMDOCKER_EXPORT_PID`, `VMDOCKER_URL`, `VMDOCKER_PRIVATE_KEY`/`HYPE_PRIVATE_KEY`/`PRV_KEY` |
| `vmdocker module build` | `VMDOCKER_AGENT_BIN`, `VMDOCKER_URL`, and the private-key chain above (also checkout `.env`) |

### Scaffolding commands

#### `new`
Create a new hymx Go project scaffold. Initializes `go.mod` with the given module and runs `go mod tidy` in the generated project.

- **`-m/--module`** (required) — Go module name, e.g. `github.com/<user>/<pkg>`.
- `-o/--out` — output base directory. Default `.`.

```bash
hype new -m github.com/<user>/<pkg> -o ./_sandbox
```

The package name is derived from the output directory name.

#### `get`
Fetch a VMM package via the Go toolchain and mount it. (Not to be confused with `vmdocker get`, which clones the VMDocker V2 repo and builds a node.)

- **`-p/--package`** (required) — Go module path of the VMM package.

#### `vmm`
Scaffold a VM module and auto-mount it into `cmd/main.go`. Run from the generated project root (the directory containing `cmd/main.go`). Inserts imports and `s.Mount(<vmm>Schema.ModuleFormat, <vmm>.Spawn)`.

- **`-n/--name`** (required) — name of the vmm.
- **`-f/--format`** (required) — module format, e.g. `hymx.token.foo.0.0.1`.

#### `mount`
Mount an existing VM module into `cmd/main.go`. Reads the module format from `<projectDir>/<vmm>/schema/schema.go`.

- **`-n/--name`** (required) — name of the vmm.

#### `module`
Generate and mount a module by name, reading `ModuleFormat` from `<projectDir>/<name>/schema/schema.go`. SDK-saved modules land under `cmd/mod/mod-<itemId>.json`.

- **`-n/--name`** (required) — name of the module.
- `-u/--node-url` — node URL. Default `http://127.0.0.1:8080`.
- `-k/--private-key` — Ethereum ECDSA secp256k1 private key hex (`0x`-prefixed).

#### `run`
Run the generated project. Executes `cd cmd && go run ./ [--mode <mode>]` from the project root.

- `-m/--mode` — start mode, `normal` or `rebuild`. Default `normal`.

### vmdocker commands

#### `vmdocker get`
Clone `https://github.com/cryptowizard0/vmdockerv2.git` and build `./build/hymx-node`.

- `--ref` — Git branch, tag, or reachable commit. Default `main`. (Commit SHAs must be reachable from an advertised remote ref.)
- `--dir` — target clone directory. Default `./vmdockerv2`.

If the checkout already exists, `hype` verifies it is a VMDocker V2 repo, fetches the ref, refuses to switch when tracked files are dirty, and rebuilds `build/hymx-node` when needed.

```bash
hype vmdocker get
hype vmdocker get --dir ./_sandbox/vmdockerv2 --ref feature/profile
```

#### `vmdocker init`
Start local services and run examples initialization. Requires `<dir>/build/hymx-node` to exist. Starts Redis via the Docker container `hype-vmdocker-redis` on port `6379`, starts the node with `./build/hymx-node start --config ./cmd/config.yaml`, waits for `http://127.0.0.1:8080/info`, then injects the parsed `.env` into `go run ./examples init`.

- `--dir` — VMDocker checkout directory. Default `./vmdockerv2`.
- **`--env-file`** (required) — `.env` file for `examples init`. Must include `VMDOCKER_PRIVATE_KEY`; `VMDOCKER_URL=http://127.0.0.1:8080` is injected automatically.

```bash
hype vmdocker init --env-file ./local.env
```

#### `vmdocker profile init`
Create a minimal profile scaffold derived from `vmdockerv2/testagent`. Writes `profile.toml`, `bin/.keep`, `skills/soul.md`, `persona/style.md`. Refuses to write into a non-empty directory.

- **`--dir`** (required) — target agent profile directory.
- **`--from`** (required) — full base image name used as `FROM` in `profile.toml`.

```bash
hype vmdocker profile init --dir ./agent --from docker/sandbox-templates:claude-code
```

#### `vmdocker module build`
Delegate module creation to VMDocker V2: runs `go run ./module --profile <profile> --agent-bin <agent>` in the checkout's `cmd/` directory, streams its output, then moves generated `cmd/mod-<itemId>.json` into `cmd/mod/` so the locally started node can load it. `hype` does not build or download the agent binary.

- **`--profile`** (required) — path to `profile.toml`.
- `--agent-bin` — path to the `vmdocker-agent` binary. Falls back to `VMDOCKER_AGENT_BIN`.
- `--dir` — VMDocker V2 checkout. Default `./vmdockerv2`.
- `--node-url` — precedence: flag → `VMDOCKER_URL` → checkout `.env` → `http://127.0.0.1:8080`.
- `--private-key` — precedence: flag → `VMDOCKER_PRIVATE_KEY` → `HYPE_PRIVATE_KEY` → `PRV_KEY` → checkout `.env`.

```bash
hype vmdocker module build \
  --dir ./vmdockerv2 --profile ./agent/profile.toml \
  --agent-bin ./bin/vmdocker-agent --private-key 0x...
```

#### `vmdocker spawn`
Spawn a process from a module. Prints the pid only.

- **`-m/--module-id`** (required) — module ID. Env: `VMDOCKER_MODULE_ID`.
- **`-s/--scheduler`** (required) — scheduler address. Env: `VMDOCKER_SCHEDULER`.
- **`-k/--private-key`** (required) — signer key. Env chain as above.
- `--runtime-type` — runtime type. Env: `RUNTIME_TYPE`.
- `--runtime-backend` — `docker` or `sandbox`. Env: `RUNTIME_BACKEND`.
- `--env KEY=VALUE` — repeatable container environment assignment.
- `-u/--node-url` — node URL. Env: `VMDOCKER_URL`; default `http://127.0.0.1:8080`.
- `--json` — JSON output.

**Tag mapping:** `--runtime-type claude` → `Container-Env-RUNTIME_TYPE=claude`; `--env TOKEN=a=b` → `Container-Env-TOKEN=a=b`; `--runtime-backend docker` → `Runtime-Backend=docker`. `RUNTIME_TYPE` is reserved for `--runtime-type`; env keys must match `^[A-Za-z_][A-Za-z0-9_]*$`; duplicate `--env` keys are rejected.

```bash
hype vmdocker spawn \
  --module-id <id> --scheduler <address> \
  --runtime-type claude --runtime-backend sandbox \
  --env TOKEN=a=b --private-key 0x...
```

#### `vmdocker export`
Send `Action=Export` to a running process and print `export ok, module id: <id>`. The returned ID can be passed directly to `vmdocker spawn --module-id <id>`.

- **`-p/--pid`** (required) — process ID. Env: `VMDOCKER_EXPORT_PID`.
- **`-k/--private-key`** (required) — signer key. Env chain as above.
- `-u/--node-url` — node URL. Env: `VMDOCKER_URL`; default `http://127.0.0.1:8080`.
- `--json` — JSON output.

#### `vmdocker clean`
Stop the local node process recorded by `cmd/hymx-*.lock`, remove `cmd/hymx-*.lock`, `cmd/hymx_*.log`, `cmd/hymx-node`, and `cmd/mod/*.json`, and remove the managed Redis container. Does **not** remove `build/hymx-node`, root `mod/*.json`, profiles, agents, or the checkout.

- `--dir` — VMDocker V2 checkout. Default `./vmdockerv2`.

### Data commands

#### `db-import`
Import a JSONL data file into Redis, calling `IDB.Commit` for each item. Each line is `{pid, nonce, msg, assign}`; items are sorted by `nonce` and must start at 0 with continuous increments.

- **`-r/--redis-url`** (required) — e.g. `redis://@localhost:6379/0`.
- **`-f/--file`** (required) — path to a `.jsonl` file.
- `-F/--force` — write even if `msg.Id` already exists (otherwise skip duplicates).

```bash
hype db-import --redis-url redis://@localhost:6379/0 --file ./data.jsonl
```

#### `db-export`
Export process data from Redis into JSONL (supports `.gz`).

- **`-r/--redis-url`** (required) — Redis connection URL.
- **`-o/--out`** (required) — output file (`.jsonl` / `.jsonl.gz`), or a directory when `--pid` is empty.
- `-p/--pid` — process id. **Optional**: leave empty to export all processes.
- `--progress-every` — print progress every N lines. Default `1000`.

```bash
hype db-export --redis-url redis://@localhost:6379/0 --pid process-123 --out ./process-123.jsonl.gz
hype db-export --redis-url redis://@localhost:6379/0 --out ./exports   # all processes -> directory
```

### Interactive & other

#### REPL
Start with `hype` (no args) or `hype repl`. Preset a key for networked commands with `HYPE_PRIVATE_KEY=0x... hype`.

- Type subcommands directly, without the `hype` prefix — e.g. `version`, `new -m ...`, `db-export ...`.
- Omitted required flags are prompted for interactively (one by one).
- `help` / `?` show `hype --help`; `exit` / `quit` / Ctrl-D exit.
- Prefix a line with `!` to run it via `bash`, e.g. `!ls`.

#### `version`
Print version information (`hype version`, `hype --version`, or `hype -v`), including the hype version, embedded hymx node version, and Go build details.

---

## Scaffolded Project Layout

`hype new` generates a project under `<out>/<pkg>/`:

```
<out>/<pkg>/
├── cmd/
│   ├── main.go
│   ├── flags.go
│   ├── const.go
│   ├── cmds.go
│   ├── cfgchainkit.go
│   ├── cfgnode.go
│   ├── cfgpay.go
│   ├── config.yaml
│   ├── config_chainkit.yaml
│   ├── config_payment.yaml
│   ├── config_test_network.yaml
│   └── mod/
│       ├── *.json                 # copied from templates (.tmpl suffix removed)
│       └── mod-<itemId>.json      # when an SDK-based module save is performed
└── <pkg>/<pkg>.go                 # interface file
```

Build the generated project from its root:

```bash
cd <out>/<pkg>
go build -o ./<pkg> ./cmd
```

Scaffold dependencies (e.g. `github.com/spf13/viper`, `github.com/urfave/cli/v2`, `github.com/hymatrix/hymx`) are fetched via `go mod tidy` during generation.

---

## Build from Source & Release

### Build

```bash
make build                        # -> build/hype
# or with go directly:
go build -o build/hype ./cmd/hype
```

### Install from source

```bash
go install ./cmd/hype     # or: make install
```

### Release flow (maintainers)

- Push a `v*` tag to trigger GoReleaser and upload GitHub Release assets.
- Publishing a GitHub Release triggers the npm publish job for `@hymx/hype`, which first verifies that `checksums.txt` and all supported tarballs are reachable.
- Configure `NPM_TOKEN` in GitHub Actions secrets for npm publishing.

---

## Notes

- **`hype get` vs `vmdocker get`** are different commands: `hype get` pulls a VMM package via go tooling; `vmdocker get` clones the VMDocker V2 repo and builds a node.
- `hype` never writes module IDs or process IDs into `.env`.
