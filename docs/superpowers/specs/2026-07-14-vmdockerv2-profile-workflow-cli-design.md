# VMDocker V2 Profile Workflow CLI Design

## Goal

Extend the `hype vmdocker` CLI to support the complete VMDocker V2 profile workflow:

1. fetch and build `vmdockerv2`;
2. initialize the local node and its dependencies;
3. scaffold a profile directory;
4. build and upload a module from `profile.toml`;
5. spawn a process from any module ID;
6. export a running process as a new module.

The exported module is started with the same `spawn` command. There is no separate `respawn` command.

## Scope and compatibility

This change replaces the current VMDocker V1 integration. `hype vmdocker` will point directly to
`github.com/cryptowizard0/vmdockerv2`; the old repository and workflow will not remain selectable.

The work is CLI-only. The embedded Web UI will not be updated. Its existing VMDocker Get action still
passes `--version`, so it will stop working after the CLI replaces that flag with `--ref`. This is an
accepted compatibility break and is not a reason to retain a deprecated CLI alias.

The following are also out of scope:

- building or downloading `vmdocker_agent`;
- runtime-specific profile presets;
- automatically writing module IDs or process IDs into `.env`;
- a one-shot roundtrip orchestration command;
- new node start/stop commands;
- Web UI forms for the new commands.

## Architecture

Use a thin orchestration design that follows the existing `hype` boundary:

- repository lifecycle and VMDocker-owned executables run through `internal/vmdocker.CommandRunner`;
- generic Hymx runtime operations use the SDK directly inside `hype`;
- local scaffolding is implemented inside `hype`.

This is approach 1A from the design discussion. It deliberately reuses VMDocker V2's executable entry
points instead of importing its internal build packages or copying profile build logic into `hype`.

The command tree is:

```text
hype vmdocker
├── get
├── init
├── profile
│   └── init
├── module
│   └── build
├── spawn
└── export
```

The existing `hype claude spawn` and `hype openclaw spawn` commands remain unchanged as
runtime-specific convenience commands.

## Commands

### `hype vmdocker get`

```text
hype vmdocker get [--dir ./vmdockerv2] [--ref main]
```

`get` clones `github.com/cryptowizard0/vmdockerv2` when the target does not exist. For an existing
checkout, it verifies that `origin` is a recognized HTTPS or SSH form of that repository. It fetches
the requested branch, tag, or commit and checks out the resolved commit in detached-HEAD mode.

Before switching an existing checkout, `get` refuses to continue when tracked files contain local
changes. It then follows the existing build pipeline: run `go mod tidy`, then build `./cmd` as
`build/hymx-node`. A checkout already at the resolved commit with a usable node binary may skip the
build.

The default ref is `main`; there is no semver discovery and no `--version` alias.

### `hype vmdocker init`

```text
hype vmdocker init [--dir ./vmdockerv2] --env-file <path>
```

`init` preserves the current behavior while operating on VMDocker V2:

1. validate the built node and the provided environment file;
2. prepare or reuse local Redis;
3. start or reuse the node daemon;
4. wait for the node health endpoint;
5. run `go run ./examples init` in the VMDocker V2 checkout.

The command does not add separate lifecycle controls.

### `hype vmdocker profile init`

```text
hype vmdocker profile init --dir <agent-dir> --from <base-image>
```

`profile init` creates this structure:

```text
<agent-dir>/
├── profile.toml
├── bin/
│   └── .keep
├── skills/
│   └── soul.md
└── persona/
    └── style.md
```

The scaffold is based on the current `vmdockerv2/testagent` fixture. `bin/.keep` is empty;
`skills/soul.md` contains `MY-SOUL`; and `persona/style.md` contains `terse, precise`. The public
allowlist includes `~/.hermes/plugin/*`, matching the source fixture, but the scaffold does not create
that optional directory.

The generated profile contains the full fixture field set and English comments:

```toml
# Declarative recipe for a vmdockerv2 agent module.
#   [dockerfile] -> input to the standardized Dockerfile generator
#   [vmdocker]   -> public allowlist used by runtime Export/Import
# The two sections are independent.

[dockerfile]
# Full base image name, used verbatim as Dockerfile FROM (no alias mapping).
# RUNTIME_TYPE does not belong here. It is passed at spawn time through the
# Container-Env-RUNTIME_TYPE tag and controls the adapter readiness check.
FROM = "<value supplied by --from>"

# Directory containing user executables. The whole directory is copied to
# /usr/local/bin and made executable. Required; it may be empty when kept by .keep.
bin = "bin"

# Optional startup command using Dockerfile CMD syntax. The adapter remains the
# ENTRYPOINT and runs this command. Arrays use exec form; strings use shell form.
# No-op modules can omit CMD.
# CMD = ["your-engine", "--serve"]

# Optional cross-distribution tool packages installed during the image build.
tools = []

# Optional Dockerfile RUN bodies. Values do not include the leading "RUN ".
RUN = []

# CMD = ["openclaw", "gateway", "--serve"]

[vmdocker]
# Export allowlist relative to HOME. Export preserves these paths, and spawn
# overlays them into a fresh workspace.
#   "~/directory/*" selects a directory recursively; "~/file" selects one file.
# Everything under HOME that is not listed remains private and is never exported.
public = ["~/skills/*", "~/persona/*", "~/.hermes/plugin/*"]
```

`--from` is required and replaces only the displayed placeholder. The command fails rather than
overwriting a regular file or a non-empty directory.

### `hype vmdocker module build`

```text
hype vmdocker module build \
  [--dir ./vmdockerv2] \
  --profile <path> \
  [--agent-bin <path>] \
  [--node-url <url>] \
  [--private-key <key>]
```

The command validates the checkout, profile, and adapter binary, converts both input paths to absolute
paths, and runs the following command from the VMDocker V2 checkout:

```text
go run ./cmd/module --profile <absolute-profile> --agent-bin <absolute-agent-bin>
```

The adapter path falls back to `VMDOCKER_AGENT_BIN`. `hype` does not build it. The node URL and private
key are passed to the child process as `VMDOCKER_URL` and `VMDOCKER_PRIVATE_KEY`. The child retains its
native `.env` behavior, including `VMDOCKER_ENV_FILE` and the checkout-root `.env` fallback.

The command streams stdout and stderr so Docker build progress remains visible. It does not parse,
duplicate, or reimplement the VMDocker V2 profile builder. The final module ID remains part of the
delegated command's normal output.

### `hype vmdocker spawn`

```text
hype vmdocker spawn \
  --module-id <id> \
  --scheduler <address> \
  [--runtime-type <type>] \
  [--runtime-backend docker|sandbox] \
  [--env KEY=VALUE ...] \
  [--node-url <url>] \
  [--private-key <key>] \
  [--json]
```

`spawn` uses the existing Hymx SDK construction in `hype` and calls `SpawnAndWait`.

Tag mapping is exact:

- `--runtime-type claude` becomes `Container-Env-RUNTIME_TYPE=claude`;
- `--runtime-backend docker` becomes `Runtime-Backend=docker`;
- each `--env KEY=VALUE` becomes `Container-Env-KEY=VALUE`.

Environment keys must match `[A-Za-z_][A-Za-z0-9_]*`. Splitting occurs at the first `=`, so values may
contain additional `=` characters. Duplicate keys are rejected. `RUNTIME_TYPE` is reserved for
`--runtime-type` and is rejected inside `--env`.

The module ID, scheduler, and runtime backend fall back to `VMDOCKER_MODULE_ID`,
`VMDOCKER_SCHEDULER`, and `RUNTIME_BACKEND`. Runtime type falls back to `RUNTIME_TYPE`. The private key
uses the existing precedence `--private-key`, `VMDOCKER_PRIVATE_KEY`, `HYPE_PRIVATE_KEY`, then
`PRV_KEY`. The node URL falls back to `VMDOCKER_URL`, then `http://127.0.0.1:8080`.

Human output reports the successful process ID. JSON output uses the existing structured runtime
result style. Neither form echoes container environment values.

### `hype vmdocker export`

```text
hype vmdocker export \
  --pid <pid> \
  [--node-url <url>] \
  [--private-key <key>] \
  [--json]
```

`export` uses `SendMessageAndWait` with `Action=Export`. The SDK response's `Message` field contains a
JSON-encoded VMM result. `hype` decodes only the required `Data` and `Error` fields, avoiding a direct
dependency on VMDocker V2 internal packages. A non-empty `Error`, malformed JSON, or empty `Data` is a
command failure. Successful `Data` is the exported module ID.

The pid falls back to `VMDOCKER_EXPORT_PID`. Node URL and private-key precedence match `spawn`.
Human and JSON output report the module ID without modifying `.env`.

## Component changes

### CLI layer

`internal/cli/vmdocker.go` owns the Cobra command tree, flag hydration, input validation, human output,
and JSON output. Shared SDK creation and result-printing helpers should be reused rather than copied.

### VMDocker manager

`internal/vmdocker` continues to own checkout, local process, and external-command behavior. It gains
focused methods for profile scaffolding and delegated module builds. The existing command runner gains
a streaming execution operation while retaining a fakeable boundary for unit tests.

The manager does not acquire SDK responsibilities and does not parse profile TOML.

### Template ownership

The English profile and two sample files are embedded project assets in `hype`, derived from the
current `vmdockerv2/testagent` fixture. They are static at a given `hype` version; `profile init` does
not require a VMDocker checkout merely to obtain its template.

## Error handling

- Git origin mismatches, dirty tracked files, unresolved refs, and build failures stop `get` with the
  underlying command context preserved.
- Profile scaffolding validates all paths before writing and never merges into a non-empty directory.
- Module build validates local files before starting Docker work and returns the child exit failure
  without printing the same build log twice.
- Spawn rejects missing IDs, invalid runtime backends, malformed environment assignments, duplicate
  keys, and conflicting runtime type input before creating an SDK client.
- Export distinguishes SDK transport errors, malformed result JSON, VMM-reported errors, and an empty
  module ID.
- Success output is emitted only after the relevant operation has completed successfully.

## Testing

Default tests must not require Docker, Redis, GitHub access, or a running Hymx node.

Unit and CLI tests cover:

- VMDocker V2 repository URLs, default `main`, arbitrary `--ref`, dirty-checkout refusal, and the
  fetch/checkout/build sequence through a fake runner;
- profile scaffold paths, exact file contents, English comments, `--from` substitution, and overwrite
  refusal;
- module build working directory, absolute arguments, environment injection, output streaming, and
  child failure propagation;
- flag-over-environment precedence;
- spawn tag construction, valid values containing `=`, duplicate and invalid environment keys,
  reserved `RUNTIME_TYPE`, and backend validation;
- export success, VMM error, malformed JSON, and empty module ID;
- human and JSON output without secret-value echoing;
- unchanged Claude and OpenClaw runtime tests.

Validation is `go test ./...` from the `hype` repository. The known Web UI VMDocker Get break is not
fixed or covered as a success criterion for this CLI-only change.

## Success criteria

The design is implemented successfully when:

1. a user can fetch and initialize VMDocker V2 through `hype vmdocker`;
2. `profile init` reproduces the agreed English `testagent` scaffold without overwriting user files;
3. `module build` delegates the authoritative build and upload path to `vmdockerv2/cmd/module`;
4. generic spawn accepts the defined runtime tags and returns a process ID;
5. export returns a new module ID that can be passed back to the same spawn command;
6. none of these operations writes IDs into user configuration files;
7. the `hype` Go test suite passes without external infrastructure.
