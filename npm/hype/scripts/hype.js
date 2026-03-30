#!/usr/bin/env node

const fs = require("fs");
const { spawn } = require("child_process");

const { getBinaryPath } = require("./lib");

const binaryPath = getBinaryPath();

if (!fs.existsSync(binaryPath)) {
  console.error(
    [
      "[hype] The hype binary is missing from this npm installation.",
      "Reinstall the package or install from source with:",
      "go install github.com/hymatrix/hype/cmd/hype@latest"
    ].join(" ")
  );
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: "inherit"
});

child.on("error", (error) => {
  console.error(`[hype] Failed to launch ${binaryPath}: ${error.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }

  process.exit(code === null ? 1 : code);
});
