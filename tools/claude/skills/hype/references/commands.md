# Hype Commands Reference

## Project Scaffolding
### `hype new`
Create a new Golang project scaffold for hymx Node.
- `--module, -m`: Go module name (e.g., `github.com/user/pkg`). **Required**.
- `--out, -o`: Output base directory. Default: `.`.

## Module Management
### `hype vmm`
Manage or scaffold a VM module and auto-mount it into `cmd/main.go`.
- `--name, -n`: Name of the VMM.
- `--format, -f`: Module format of the VMM.

### `hype mount`
Mount an existing VM module into `cmd/main.go`.
- `--name, -n`: Name of the VMM.

### `hype module`
Generate and mount a module by name.
- `--name, -n`: Name of the module.
- `--node-url, -u`: Node URL. Default: `http://127.0.0.1:8080`.
- `--private-key, -k`: Private key.

## Data Operations
### `hype db-import`
Import JSONL data file into Redis.
- `--redis-url, -r`: Redis connection URL. **Required**.
- `--file, -f`: Path to JSONL file. **Required**.
- `--force, -F`: Write even if msg.Id exists.

### `hype db-export`
Export a process from Redis into JSONL (supports .gz).
- `--redis-url, -r`: Redis connection URL. **Required**.
- `--pid, -p`: Process ID. **Required**.
- `--out, -o`: Output file path. **Required**.
- `--progress-every`: Print progress every N lines.

## Execution
### `hype run`
Run the generated project.
- `--mode, -m`: Start mode (`normal` or `rebuild`). Default: `normal`.

### `hype repl`
Enter interactive shell mode.
- Type subcommands directly (no `hype` prefix).
- Use `!` for shell escape.
- `help` or `?` for help.
- `exit` or `quit` to exit.
