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
  },
  {
    TYPE: 'Linux',
    ARCHITECTURE: 'x64',
    TARGET: 'fleetctl-linux.tar.gz',
  },
  {
    TYPE: 'Darwin',
    ARCHITECTURE: 'x64',
    TARGET: 'fleetctl-macos.tar.gz',
  },
];

const getPlatform = () => {
  const type = os.type();
  const architecture = os.arch();

  for (const index in supportedPlatforms) {
    const supportedPlatform = supportedPlatforms[index];
    if (
      type === supportedPlatform.TYPE &&
        architecture === supportedPlatform.ARCHITECTURE
    ) {
      return supportedPlatform.TARGET;
    }
  }

  console.error(
    `Platform ${type} and architecture ${architecture} is not supported.`,
  );
  process.exit(1);

  return undefined;
};

const getBinary = () => {
  const target = getPlatform();
  // the url for this binary is constructed from values in `package.json`
  const url = `${downloadUrl}/${version}/${target}`;
  return new Binary('fleetctl', url);
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
