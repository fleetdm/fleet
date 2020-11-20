// Based on the MIT licensed example in
// https://github.com/cloudflare/binary-install/blob/master/packages/binary-install-example/binary.js

const { Binary } = require('binary-install');
const os = require('os');

const { version } = require('./package.json');

const downloadUrl = 'https://github.com/fleetdm/fleet/releases/download';

const supportedPlatforms = [
  {
    TYPE: 'Windows_NT',
    ARCHITECTURE: 'x64',
    TARGET: 'fleetctl-windows.tar.gz',
    BINARY_NAME: 'fleetctl.exe',
  },
  {
    TYPE: 'Linux',
    ARCHITECTURE: 'x64',
    TARGET: 'fleetctl-linux.tar.gz',
    BINARY_NAME: 'fleetctl',
  },
  {
    TYPE: 'Darwin',
    ARCHITECTURE: 'x64',
    TARGET: 'fleetctl-macos.tar.gz',
    BINARY_NAME: 'fleetctl',
  },
];

const getPlatformMetadata = () => {
  const type = os.type();
  const architecture = os.arch();

  for (const index in supportedPlatforms) {
    const supportedPlatform = supportedPlatforms[index];
    if (
      type === supportedPlatform.TYPE &&
        architecture === supportedPlatform.ARCHITECTURE
    ) {
      return supportedPlatform;
    }
  }

  console.error(
    `Platform ${type} and architecture ${architecture} is not supported.`,
  );
  process.exit(1);

  return undefined;
};

const getBinary = () => {
  const metadata = getPlatformMetadata();
  // the url for this binary is constructed from values in `package.json`
  const url = `${downloadUrl}/${version}/${metadata.TARGET}`;
  return new Binary(metadata.BINARY_NAME, url);
};

const run = () => {
  const binary = getBinary();
  binary.run();
};

const install = () => {
  const binary = getBinary();
  binary.install();
};

module.exports = {
  install,
  run,
};
