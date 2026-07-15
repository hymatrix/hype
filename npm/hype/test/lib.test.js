const test = require("node:test");
const assert = require("node:assert/strict");

const { getAssetName, getAssetUrl, getReleaseTag } = require("../scripts/lib");

test("maps supported platforms to release asset names", () => {
  assert.equal(getAssetName("darwin", "arm64"), "hype_Darwin_arm64.tar.gz");
  assert.equal(getAssetName("darwin", "x64"), "hype_Darwin_x86_64.tar.gz");
  assert.equal(getAssetName("linux", "arm64"), "hype_Linux_arm64.tar.gz");
  assert.equal(getAssetName("linux", "x64"), "hype_Linux_x86_64.tar.gz");
});

test("derives GitHub release tags from npm package versions", () => {
  assert.equal(getReleaseTag("0.1.0"), "v0.1.0");
  assert.equal(getReleaseTag("v0.1.0"), "v0.1.0");
});

test("builds release download URLs from package versions", () => {
  assert.equal(
    getAssetUrl("0.1.0", "checksums.txt"),
    "https://github.com/hymatrix/hype/releases/download/v0.1.0/checksums.txt"
  );
});

test("fails clearly for unsupported targets", () => {
  assert.throws(() => getAssetName("win32", "x64"), {
    code: "UNSUPPORTED_PLATFORM",
    message: /go install github\.com\/hymatrix\/hype\/cmd\/hype@latest/
  });
});
