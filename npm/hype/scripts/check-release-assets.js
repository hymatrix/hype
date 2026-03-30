#!/usr/bin/env node

const https = require("https");

const pkg = require("../package.json");
const { CHECKSUMS_ASSET_NAME, getAssetUrl, getSupportedTargets } = require("./lib");

async function main() {
  const version = process.env.HYPE_NPM_PACKAGE_VERSION || pkg.version;
  const assetNames = [CHECKSUMS_ASSET_NAME, ...getSupportedTargets().map((target) => target.assetName)];

  for (const assetName of assetNames) {
    const url = getAssetUrl(version, assetName);
    await assertReachable(url);
    console.log(`[hype] Verified release asset: ${url}`);
  }
}

function assertReachable(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) {
      reject(new Error(`Too many redirects while checking ${url}`));
      return;
    }

    const request = https.request(
      url,
      {
        method: "HEAD",
        headers: {
          "user-agent": "@hymx/hype-release-check"
        }
      },
      (response) => {
        const statusCode = response.statusCode || 0;

        if (statusCode >= 300 && statusCode < 400 && response.headers.location) {
          const redirectedUrl = new URL(response.headers.location, url).toString();
          response.resume();
          assertReachable(redirectedUrl, redirects + 1).then(resolve, reject);
          return;
        }

        response.resume();

        if (statusCode < 200 || statusCode >= 400) {
          reject(new Error(`Asset check failed for ${url}: HTTP ${statusCode}`));
          return;
        }

        resolve();
      }
    );

    request.on("error", reject);
    request.end();
  });
}

main().catch((error) => {
  console.error(`[hype] ${error.message}`);
  process.exit(1);
});
