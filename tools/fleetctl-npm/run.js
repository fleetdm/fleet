#!/usr/bin/env node

const child = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const axios = require("axios");
const rimraf = require("rimraf");
const tar = require("tar");

const { version } = require("./package.json");

// Strip any v4.0.0-1 style suffix (but not -rc1) so that the correct package is
// downloaded if there is a mistake in the NPM publish and we need to release a
// -1, etc. (because NPM packages are immutable and can't be fixed after a mistake).
let strippedVersion = version.replace(/-[0-9]+/i, "");
if (!strippedVersion.startsWith("v")) {
  strippedVersion = `v${strippedVersion}`;
}

const binDir = path.join(__dirname, "install");
// Determine the install directory by version so that we can detect when we need
// to upgrade to a new version.
const installDir = path.join(binDir, strippedVersion);

const platform = (() => {
  switch (os.type()) {
    case "Windows_NT":
      return "windows";
    case "Linux":
      return "linux";
    case "Darwin":
      return "macos";
    default:
      throw new Error(`platform ${os.type} unrecognized`);
  }
})();

const binName = platform === "windows" ? "fleetctl.exe" : "fleetctl";
const binPath = path.join(installDir, binName);

const install = async () => {
  const url = `https://github.com/fleetdm/fleet/releases/download/fleet-${strippedVersion}/fleetctl_${strippedVersion}_${platform}.tar.gz`;

  fs.mkdirSync(installDir, { recursive: true });

  try {
    const response = await axios({ url, responseType: "stream" });

    // Strip the outer directory when extracting. Just get the binary.
    const tarWriter = tar.extract({ strip: 1, cwd: installDir });
    response.data.pipe(tarWriter);

    // Need to return a promise with the writer to ensure we can await for it to complete.
    return new Promise((resolve, reject) => {
      tarWriter.on("finish", resolve);
      tarWriter.on("error", reject);
    });
  } catch (err) {
    throw new Error(`download archive ${url}: ${err.message}`);
  }
};

const run = async () => {
  if (!fs.existsSync(binPath)) {
    // Remove any existing binaries before installing the new one.
    rimraf.sync(binDir);
    console.log(`Installing fleetctl ${strippedVersion}...`);
    try {
      await install();
    } catch (err) {
      console.error(`Failed to install: ${err.message}`);
      process.exit(1);
    }
    console.log("Install completed.");
  }

  const [, , ...args] = process.argv;
  const options = { cwd: process.cwd(), stdio: "inherit" };
  const { status, error } = child.spawnSync(binPath, args, options);

  if (error) {
    console.error(error);
    process.exit(1);
  }

  process.exit(status);
};

run();
