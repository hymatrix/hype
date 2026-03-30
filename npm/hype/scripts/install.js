#!/usr/bin/env node

const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const { spawnSync } = require("child_process");

const pkg = require("../package.json");
const { BINARY_NAME, getAssetName, getAssetUrl, getBinaryPath, getRuntimeDir } = require("./lib");

async function main() {
  const assetName = getAssetName(process.platform, process.arch);
  const assetUrl = getAssetUrl(pkg.version, assetName);
  const runtimeDir = getRuntimeDir();
  const binaryPath = getBinaryPath();
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "hype-npm-"));
  const archivePath = path.join(tmpDir, assetName);

  try {
    console.log(`[hype] Downloading ${assetName} from ${assetUrl}`);
    fs.rmSync(runtimeDir, { recursive: true, force: true });
    fs.mkdirSync(runtimeDir, { recursive: true });

    await downloadToFile(assetUrl, archivePath);
    extractArchive(archivePath, runtimeDir);
    finalizeBinary(runtimeDir, binaryPath);

    fs.chmodSync(binaryPath, 0o755);
    console.log(`[hype] Installed ${BINARY_NAME} to ${binaryPath}`);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

function finalizeBinary(runtimeDir, binaryPath) {
  if (fs.existsSync(binaryPath)) {
    return;
  }

  const discovered = findFile(runtimeDir, BINARY_NAME);
  if (!discovered) {
    throw new Error(`Archive did not contain a ${BINARY_NAME} binary.`);
  }

  fs.renameSync(discovered, binaryPath);
}

function findFile(rootDir, fileName) {
  const stack = [rootDir];

  while (stack.length > 0) {
    const currentDir = stack.pop();
    const entries = fs.readdirSync(currentDir, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(currentDir, entry.name);

      if (entry.isDirectory()) {
        stack.push(fullPath);
        continue;
      }

      if (entry.isFile() && entry.name === fileName) {
        return fullPath;
      }
    }
  }

  return null;
}

function extractArchive(archivePath, runtimeDir) {
  const result = spawnSync("tar", ["-xzf", archivePath, "-C", runtimeDir], {
    encoding: "utf8"
  });

  if (result.status !== 0) {
    throw new Error(
      `Failed to extract ${path.basename(archivePath)} with tar: ${result.stderr || result.stdout || "unknown error"}`
    );
  }
}

function downloadToFile(url, destination, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) {
      reject(new Error(`Too many redirects while downloading ${url}`));
      return;
    }

    const request = https.get(
      url,
      {
        headers: {
          "user-agent": "@hymx/hype-installer"
        }
      },
      (response) => {
        const statusCode = response.statusCode || 0;

        if (statusCode >= 300 && statusCode < 400 && response.headers.location) {
          const redirectedUrl = new URL(response.headers.location, url).toString();
          response.resume();
          downloadToFile(redirectedUrl, destination, redirects + 1).then(resolve, reject);
          return;
        }

        if (statusCode !== 200) {
          response.resume();
          reject(new Error(`Download failed for ${url}: HTTP ${statusCode}`));
          return;
        }

        const file = fs.createWriteStream(destination);
        response.pipe(file);

        file.on("finish", () => {
          file.close((error) => {
            if (error) {
              reject(error);
              return;
            }
            resolve();
          });
        });

        file.on("error", (error) => {
          fs.rmSync(destination, { force: true });
          reject(error);
        });
      }
    );

    request.on("error", reject);
  });
}

main().catch((error) => {
  console.error(`[hype] Install failed: ${error.message}`);
  process.exit(1);
});
