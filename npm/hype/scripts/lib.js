const path = require("path");

const REPOSITORY = "hymatrix/hype";
const RELEASE_HOST = "https://github.com";
const BINARY_NAME = "hype";
const CHECKSUMS_ASSET_NAME = "checksums.txt";

const TARGET_ASSETS = Object.freeze({
  "darwin:arm64": "hype_Darwin_arm64.tar.gz",
  "darwin:x64": "hype_Darwin_x86_64.tar.gz",
  "linux:arm64": "hype_Linux_arm64.tar.gz",
  "linux:x64": "hype_Linux_x86_64.tar.gz"
});

function getReleaseTag(version) {
  if (!version) {
    throw new Error("Package version is required");
  }

  return version.startsWith("v") ? version : `v${version}`;
}

function getAssetName(platform, arch) {
  const key = `${platform}:${arch}`;
  const assetName = TARGET_ASSETS[key];

  if (!assetName) {
    const supported = Object.keys(TARGET_ASSETS)
      .map((target) => target.replace(":", "/"))
      .join(", ");
    const error = new Error(
      [
        `Unsupported platform for @hymx/hype: ${platform}/${arch}.`,
        `Supported targets: ${supported}.`,
        "Install from source instead:",
        "go install github.com/hymatrix/hype/cmd/hype@latest"
      ].join(" ")
    );
    error.code = "UNSUPPORTED_PLATFORM";
    throw error;
  }

  return assetName;
}

function getSupportedTargets() {
  return Object.keys(TARGET_ASSETS).map((key) => {
    const [platform, arch] = key.split(":");
    return {
      platform,
      arch,
      assetName: TARGET_ASSETS[key]
    };
  });
}

function getAssetUrl(version, assetName) {
  return `${RELEASE_HOST}/${REPOSITORY}/releases/download/${getReleaseTag(version)}/${assetName}`;
}

function getRuntimeDir(baseDir = path.join(__dirname, "..")) {
  return path.join(baseDir, "runtime");
}

function getBinaryPath(baseDir = path.join(__dirname, "..")) {
  return path.join(getRuntimeDir(baseDir), BINARY_NAME);
}

module.exports = {
  BINARY_NAME,
  CHECKSUMS_ASSET_NAME,
  REPOSITORY,
  TARGET_ASSETS,
  getAssetName,
  getAssetUrl,
  getBinaryPath,
  getReleaseTag,
  getRuntimeDir,
  getSupportedTargets
};
