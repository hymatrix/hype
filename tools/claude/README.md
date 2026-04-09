# Claude Commands in Hype

This document explains how to use Claude-related commands in [`hype`](/Users/webbergao/work/src/HymxWorkspace/hype/README.md).

It covers two paths:

- command line
- HypeUI

## Prerequisites

Before using Claude commands, make sure all of the following are ready:

- `vmdocker` node is running and reachable, usually `http://127.0.0.1:8080`
- a Claude-enabled `vmdocker_agent` module has already been generated
- you know the Claude `module id` and `scheduler`
- you have a valid `ANTHROPIC_API_KEY`
- if needed, you also have `ANTHROPIC_BASE_URL` and `ANTHROPIC_MODEL`

Recommended environment variables:

```bash
export HYPE_PRIVATE_KEY='your_private_key'
export VMDOCKER_PRIVATE_KEY="$HYPE_PRIVATE_KEY"

export VMDOCKER_MODULE_ID='your_claude_module_id'
export VMDOCKER_SCHEDULER='your_scheduler'
export RUNTIME_BACKEND='docker'   # or sandbox

export ANTHROPIC_API_KEY='your_api_key'
export ANTHROPIC_BASE_URL='https://your-base-url'
export ANTHROPIC_MODEL='your-model'
export CLAUDE_CODE_FLAGS=''
```

## Command Line

### 1. Spawn a Claude Process

Create a Claude runtime process:

```bash
cd /Users/webbergao/work/src/HymxWorkspace/hype

go run ./cmd/hype claude spawn \
  --module-id "$VMDOCKER_MODULE_ID" \
  --scheduler "$VMDOCKER_SCHEDULER" \
  --api-key "$ANTHROPIC_API_KEY" \
  --base-url "$ANTHROPIC_BASE_URL" \
  --model "$ANTHROPIC_MODEL" \
  --runtime-backend "$RUNTIME_BACKEND" \
  --json
```

Expected result:

- `action=spawn`
- `pid=<new process id>`

Keep the returned `pid` for the next commands.

### 2. Chat with Claude

Send a chat-style request:

```bash
go run ./cmd/hype claude chat \
  --pid '<pid>' \
  --command '你好，请介绍一下自己' \
  --json
```

About `--command`:

- it is just the prompt text sent to Claude
- natural language is enough
- no special DSL is required

Examples:

```bash
go run ./cmd/hype claude chat --pid '<pid>' -c '请只回复 OK' --json
go run ./cmd/hype claude chat --pid '<pid>' -c '查看当前目录并总结项目结构' --json
```

### 3. Run a Generic Claude Prompt

Use the explicit `exec` subcommand:

```bash
go run ./cmd/hype claude exec \
  --pid '<pid>' \
  -p '请只回复 OK' \
  --json
```

You can also use the shortcut form:

```bash
go run ./cmd/hype claude \
  --pid '<pid>' \
  -p '请只回复 OK' \
  --json
```

Both forms send a prompt to the running Claude process.

### 4. When to Use `chat` vs `exec`

- use `chat` when the interaction is conversational
- use `exec` or `claude -p` when you want a generic one-shot prompt

At the current runtime layer, both paths end up calling Claude Code with prompt text. The main difference is the action label recorded in the result.

### 5. Common Errors

`runtime type not supported: claude`

- the module still points to an old `vmdocker_agent` image

`private-key is required`

- `HYPE_PRIVATE_KEY` or `VMDOCKER_PRIVATE_KEY` is missing

`api-key is required`

- `ANTHROPIC_API_KEY` is missing

`402` or credits-related API error

- provider-side credits or quota is insufficient

## HypeUI

### 1. Start the UI

```bash
cd /Users/webbergao/work/src/HymxWorkspace/hype
go run ./cmd/hype ui
```

Open:

```text
http://127.0.0.1:7788
```

### 2. Import `.env`

In the top `.env Import` panel, import a file or load a path.

Relevant keys for Claude:

- `VMDOCKER_MODULE_ID`
- `VMDOCKER_SCHEDULER`
- `RUNTIME_BACKEND`
- `VMDOCKER_PRIVATE_KEY`
- `ANTHROPIC_API_KEY`
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_MODEL`
- `CLAUDE_CODE_FLAGS`

The imported values:

- stay in memory only
- prefill matching form fields
- are not stored in browser local storage

### 3. Spawn in UI

In the left navigation:

- choose root command `claude`
- choose command `spawn`

Fill or import:

- `module-id`
- `scheduler`
- `api-key`
- optional `base-url`
- optional `model`
- optional `code-flags`
- optional `runtime-backend`

Run the command. After success:

- the result panel will show the parsed response
- the `Spawned PIDs` panel will show the new pid

### 4. Chat in UI

Switch to:

- `claude / chat`

Provide:

- `pid`
- `command`

Then run it.

The result panel will show:

- parsed JSON when available
- `reply` when present
- raw stdout/stderr

### 5. Generic Prompt in UI

Switch to:

- `claude / exec`

Provide:

- `pid`
- `prompt`

This is the UI equivalent of:

```bash
hype claude exec --pid '<pid>' -p '...'
```

HypeUI intentionally exposes `claude / exec`, not the root shortcut `claude -p`.

## Quick Test Flow

If you want the shortest end-to-end manual test:

1. Run `claude spawn`
2. Copy the returned `pid`
3. Run `claude chat`
4. Run `claude exec`

Example:

```bash
go run ./cmd/hype claude spawn --module-id "$VMDOCKER_MODULE_ID" --scheduler "$VMDOCKER_SCHEDULER" --api-key "$ANTHROPIC_API_KEY" --json
go run ./cmd/hype claude chat --pid '<pid>' -c '你好，请介绍一下自己' --json
go run ./cmd/hype claude exec --pid '<pid>' -p '请只回复 OK' --json
```
