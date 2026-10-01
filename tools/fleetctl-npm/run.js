#!/usr/bin/env node
/* eslint-disable @typescript-eslint/no-var-requires */

const { spawnSync } = require("child_process");
const {
  chmodSync,
  existsSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  renameSync,
  rmSync,
} = require("fs");
const { constants: osConstants, type } = require("os");
const { join } = require("path");
const { arch } = require("process");
const { pipeline } = require("stream/promises");

const axios = require("axios");
const { rimrafSync } = require("rimraf");
const { extract } = require("tar");
const { version } = require("./package.json");

// Fail closed: if anything below leaves a promise pending (e.g. a download stream that stops
// without an error event), the event loop drains and Node would otherwise exit 0 silently.
process.exitCode = 1;

// Strip any v4.0.0-1 style suffix (but not -rc1) so that the correct package is
// downloaded if there is a mistake in the NPM publish and we need to release a
// -1, etc. (because NPM packages are immutable and can't be fixed after a mistake).
let strippedVersion = version.replace(/-[0-9]+/i, "");
if (!strippedVersion.startsWith("v")) {
  strippedVersion = `v${strippedVersion}`;
}

const binDir = join(__dirname, "install");
// Determine the install directory by version so that we can detect when we need
// to upgrade to a new version.
const installDir = join(binDir, strippedVersion);

const platform = (() => {
  switch (type()) {
    case "Windows_NT":
      return `windows_${arch === "arm64" ? "arm64" : "amd64"}`;
    case "Linux":
      return `linux_${arch === "arm64" ? "arm64" : "amd64"}`;
    case "Darwin":
      return "macos";
    default:
      throw new Error(`platform ${type()} unrecognized`);
  }
})();

const binName = platform.startsWith("windows") ? "fleetctl.exe" : "fleetctl";
const binPath = join(installDir, binName);

const install = async () => {
  const url = `https://github.com/fleetdm/fleet/releases/download/fleet-${strippedVersion}/fleetctl_${strippedVersion}_${platform}.tar.gz`;

  mkdirSync(binDir, { recursive: true });
  // Extract into a temporary directory and rename it into place, so an interrupted
  // download never leaves a partial binary where the next run would find it.
  const tmpDir = mkdtempSync(join(binDir, "tmp-"));

  try {
    let response;
    try {
      response = await axios({ url, responseType: "stream" });
    } catch (err) {
      if (axios.isAxiosError(err)) {
        throw new Error(`download archive ${url}: ${err.message}`);
      }
      throw err;
    }

    // Unlike stream.pipe(), pipeline() rejects on errors and premature close from either side.
    // Strip the outer directory when extracting. Just get the binary.
    await pipeline(response.data, extract({ strip: 1, cwd: tmpDir }));

    // Validate before caching: a bad entry here would otherwise be reused on every later run.
    const extracted = join(tmpDir, binName);
    const stat = existsSync(extracted) ? lstatSync(extracted) : null;
    if (!stat || !stat.isFile()) {
      throw new Error(`archive ${url} does not contain a ${binName} file`);
    }
    if (!platform.startsWith("windows") && !(stat.mode & 0o111)) {
      throw new Error(`${binName} in archive ${url} is not executable`);
    }
    // mkdtempSync creates the directory as 0700; keep it traversable for other users when
    // installed as root (sudo npm install -g fleetctl), like the plain mkdirSync it replaced.
    chmodSync(tmpDir, 0o755);
    renameSync(tmpDir, installDir);
  } catch (err) {
    rmSync(tmpDir, { recursive: true, force: true });
    throw err;
  }
};

const run = async () => {
  if (!existsSync(binPath)) {
    // Remove any existing binaries before installing the new one.
    rimrafSync(binDir);
    console.log(`Installing fleetctl ${strippedVersion}...`);
    try {
      await install();
    } catch (err) {
      // Users commonly see permission errors when trying to install the binaries if they have run
      // `sudo npm install -g fleetctl` (or the Windows equivalent of running as admin), then later
      // try to run fleetctl without those elevated privileges.
      if (err.code === "EACCES") {
        switch (process.platform) {
          case "darwin":
          case "linux":
            console.error(
              "Error: It looks like your fleetctl has been installed as root."
            );
            console.error("Please re-run this command with sudo.");
            process.exit(1);
            break;
          case "win32":
          case "win64":
            console.error(
              "Error: It looks like your fleetctl has been installed as administrator."
            );
            console.error(
              "Please re-run this command using 'Run as administrator'."
            );
            process.exit(1);
            break;
          default:
          // Fall through to generic error print below
        }
      }
      console.error(`Error: Failed to install: ${err.message}`);
      process.exit(1);
    }
    console.log("Install completed.");
  }

  const [, , ...args] = process.argv;
  const options = { cwd: process.cwd(), stdio: "inherit" };
  const { status, signal, error } = spawnSync(binPath, args, options);

  if (error) {
    console.error(error);
    process.exit(1);
  }
  if (status === null) {
    // Killed by a signal: status is null and process.exit(null) would report success.
    // Mirror the shell convention (128 + signal number) and stay quiet on SIGPIPE, which is
    // routine when the output is piped to a command that exits early (e.g. `| head`).
    if (signal !== "SIGPIPE") {
      console.error(`Error: fleetctl was terminated by signal ${signal}`);
    }
    process.exit(128 + (osConstants.signals[signal] || 0));
  }

  process.exit(status);
};

run().catch((err) => {
  console.error(`Error: ${err.message}`);
  process.exit(1);
});
