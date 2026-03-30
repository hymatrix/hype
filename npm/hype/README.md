# @hymx/hype

Install the `hype` CLI from npm without requiring a local Go toolchain.

## Install

```bash
npm install -g @hymx/hype
```

Or run it directly with:

```bash
npx @hymx/hype --version
```

## How it works

This package downloads the matching `hype` release asset from GitHub Releases during `postinstall` and exposes the `hype` binary on your `PATH`.

Supported targets:

- macOS arm64
- macOS x64
- Linux arm64
- Linux x64

If your platform is not supported by this npm package, install from source instead:

```bash
go install github.com/hymatrix/hype/cmd/hype@latest
```
