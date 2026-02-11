---
name: hype
description: "Official project scaffolding and management CLI tool for the hymx Node. Use this skill when the user wants to: (1) Create a new hymx project scaffold, (2) Manage or scaffold VMM modules, (3) Mount existing modules into a project, (4) Import/Export process data between JSONL and Redis, (5) Run or debug hymx projects."
---

# Hype

## Overview
`hype` is the primary tool for developing on the `hymx` platform. It automates the creation of project structures, boilerplate code for VM modules (VMMs), and data synchronization between local storage (Redis) and portable formats (JSONL).

## Prerequisites
Before using `hype`, ensure it is installed via `go install`:
```bash
go install github.com/hymatrix/hype/cmd/hype@latest
```

## Core Capabilities

### 1. Project Scaffolding
Use `hype new` to start a new project.
```bash
hype new -m github.com/username/my-project -o ./output
```

### 2. Module Management
Scaffold and mount new VMMs:
```bash
# Create and mount a new VMM
hype vmm --name my_vmm --format lua

# Mount an existing VMM package
hype mount --name existing_vmm
```

### 3. Data Syncing
Import data from JSONL to Redis for local replay or testing:
```bash
hype db-import --redis-url "redis://localhost:6379" --file ./data.jsonl
```

Export process data to share or archive:
```bash
hype db-export --redis-url "redis://localhost:6379" --pid process_id --out ./export.jsonl
```

## Reference Material
- **Command Details**: See [commands.md](references/commands.md) for a full list of flags and subcommands.
- **Project Layout**: See [project_structure.md](references/project_structure.md) to understand how `hype` organizes generated projects and where to find key files like `main.go`.

## Development Workflow
1. **Initialize**: Create a new project with `hype new`.
2. **Develop VMMs**: Use `hype vmm` to add new modules.
3. **Mount SDK Modules**: Use `hype module` to generate and mount modules based on SDK schemas.
4. **Test**: Run the project with `hype run` or enter the `hype repl` for interactive debugging.
5. **Sync Data**: Use `db-import` and `db-export` to manage test data.
