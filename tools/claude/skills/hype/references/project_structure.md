# Generated Project Structure

When you run `hype new`, it generates a project with the following structure:

- `cmd/`: CLI entry points and configurations.
    - `main.go`: Main entry point where VMMs are mounted.
    - `flags.go`: CLI flags definition.
    - `const.go`: Constants for the project.
    - `cmds.go`: Command logic.
    - `config.yaml`: Main configuration file.
    - `config_chainkit.yaml`: Chainkit specific config.
    - `config_payment.yaml`: Payment specific config.
    - `mod/`: Directory for module JSON files.
- `<pkg>/`: Directory named after the package.
    - `<pkg>.go`: Main interface definition for the module.

## Mounting Logic
When using `vmm`, `mount`, or `module` commands, `hype` automatically modifies `cmd/main.go` to:
1. Add necessary imports.
2. Add `s.Mount(...)` calls under specific hint comments.
