# VMDocker V2 CLI Workflow

This guide describes the Hype CLI flow for VMDocker V2 profile-based modules.

The workflow is composable:

1. Fetch and build the VMDocker V2 checkout.
2. Start the local VMDocker node if needed.
3. Create a `profile.toml` agent scaffold.
4. Build a module by delegating to VMDocker V2.
5. Spawn a process from the module.
6. Export a running process into a reusable module ID.

Hype prints IDs, but it does not write module IDs or process IDs into `.env`.

## Prerequisites

- A built `hype` binary, or use `go run ./cmd/hype`.
- Docker and Redis available for `hype vmdocker init`.
- A VMDocker agent binary path for module builds.
- A signing private key for module build, spawn, and export.
- A scheduler address for spawn.

## 1. Fetch VMDocker V2

```bash
hype vmdocker get
```

Defaults:

- repo: `https://github.com/cryptowizard0/vmdockerv2.git`
- directory: `./vmdockerv2`
- ref: `main`

Use a branch, tag, or commit with `--ref`:

```bash
hype vmdocker get --dir ./vmdockerv2 --ref main
```

If the checkout already exists, Hype verifies that it is a VMDocker V2 repo, fetches the requested ref, refuses to switch when tracked files are dirty, and rebuilds `build/hymx-node` when needed.

## 2. Initialize The Local Node

```bash
hype vmdocker init --dir ./vmdockerv2 --env-file ./local.env
```

The `.env` file must include:

```bash
VMDOCKER_PRIVATE_KEY=0x...
```

The command starts Redis, starts the built node, waits for node health, and runs VMDocker V2 examples initialization.

## 3. Create A Profile Scaffold

```bash
hype vmdocker profile init \
  --dir ./agent \
  --from docker/sandbox-templates:claude-code
```

This creates a minimal scaffold derived from `vmdockerv2/testagent`:

```text
agent/
|-- profile.toml
|-- bin/
|   `-- .keep
|-- skills/
|   `-- soul.md
`-- persona/
    `-- style.md
```

The target directory must be empty or absent.

## 4. Build A Module

```bash
hype vmdocker module build \
  --dir ./vmdockerv2 \
  --profile ./agent/profile.toml \
  --agent-bin ./bin/vmdocker-agent \
  --private-key 0x...
```

Hype delegates the build to VMDocker V2:

```bash
cd <vmdockerv2-checkout>/cmd
go run ./module --profile <profile.toml> --agent-bin <vmdocker-agent>
```

After the delegated build succeeds, Hype moves generated `cmd/mod-<itemId>.json` files into `cmd/mod/`.
This keeps the local node started by `hype vmdocker init` able to load the module from its working directory.

Input precedence:

- `--agent-bin`, then `VMDOCKER_AGENT_BIN`
- `--node-url`, then `VMDOCKER_URL`, then checkout `.env`, then `http://127.0.0.1:8080`
- `--private-key`, then `VMDOCKER_PRIVATE_KEY`, then `HYPE_PRIVATE_KEY`, then `PRV_KEY`, then checkout `.env`

Hype does not build or download the agent binary.

## 5. Spawn A Process

```bash
hype vmdocker spawn \
  --module-id <module-id> \
  --scheduler <scheduler-address> \
  --runtime-type claude \
  --runtime-backend sandbox \
  --env TOKEN=a=b \
  --private-key 0x...
```

Input precedence:

- `--module-id`, then `VMDOCKER_MODULE_ID`
- `--scheduler`, then `VMDOCKER_SCHEDULER`
- `--runtime-type`, then `RUNTIME_TYPE`
- `--runtime-backend`, then `RUNTIME_BACKEND`
- `--node-url`, then `VMDOCKER_URL`, then `http://127.0.0.1:8080`
- `--private-key`, then `VMDOCKER_PRIVATE_KEY`, then `HYPE_PRIVATE_KEY`, then `PRV_KEY`

Tag mapping:

- `--runtime-type claude` -> `Container-Env-RUNTIME_TYPE=claude`
- `--env TOKEN=a=b` -> `Container-Env-TOKEN=a=b`
- `--runtime-backend sandbox` -> `Runtime-Backend=sandbox`

`RUNTIME_TYPE` is reserved for `--runtime-type`. Duplicate `--env` keys are rejected.

Use `--json` for machine-readable output.

## 6. Export A Process

```bash
hype vmdocker export \
  --pid <pid> \
  --private-key 0x...
```

Input precedence:

- `--pid`, then `VMDOCKER_EXPORT_PID`
- `--node-url`, then `VMDOCKER_URL`, then `http://127.0.0.1:8080`
- `--private-key`, then `VMDOCKER_PRIVATE_KEY`, then `HYPE_PRIVATE_KEY`, then `PRV_KEY`

Export sends `Action=Export` to the process and prints:

```text
export ok, module id: <module-id>
```

The returned module ID can be passed directly to `hype vmdocker spawn --module-id <module-id>`.

## Maintenance: Clean The Local Runtime

Use this when you need to reset the local runtime state before re-running initialization or local tests.

```bash
hype vmdocker clean --dir ./vmdockerv2
```

The command stops the local node process recorded by `cmd/hymx-*.lock`, removes local runtime files, and removes the managed Redis container `hype-vmdocker-redis`.

It removes:

- `cmd/hymx-*.lock`
- `cmd/hymx_*.log`
- `cmd/hymx-node`
- `cmd/mod/*.json`

It does not remove:

- `build/hymx-node`
- root `mod/*.json`
- profiles, agents, or the checkout directory

## Notes

- There is no `respawn` command. Use `export`, then `spawn` with the returned module ID.
- Hype never writes module IDs or process IDs into `.env`.
- The embedded Web UI VMDocker Get action still uses the removed `--version` flag and is not part of this CLI-only V2 migration.
