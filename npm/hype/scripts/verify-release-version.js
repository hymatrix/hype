#!/usr/bin/env node

const pkg = require("../package.json");
const { getReleaseTag } = require("./lib");

const releaseTag = process.argv[2];
const expectedTag = getReleaseTag(pkg.version);

if (!releaseTag) {
  console.error("[hype] Missing release tag argument.");
  process.exit(1);
}

if (releaseTag !== expectedTag) {
  console.error(
    `[hype] npm package version ${pkg.version} does not match release tag ${releaseTag}. Expected ${expectedTag}.`
  );
  process.exit(1);
}

console.log(`[hype] npm package version ${pkg.version} matches release tag ${releaseTag}.`);
