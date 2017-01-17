export const platformIconClass = (platform = '') => {
  if (!platform) return false;

  const lowerPlatform = platform.toLowerCase();

  switch (lowerPlatform) {
    case 'macos': return 'apple';
    case 'mac os x': return 'apple';
    case 'mac osx': return 'apple';
    case 'mac os': return 'apple';
    case 'darwin': return 'apple';
    case 'apple': return 'apple';
    case 'centos': return 'centos';
    case 'centos linux': return 'centos';
    case 'ubuntu': return 'ubuntu';
    case 'ubuntu linux': return 'ubuntu';
    case 'linux': return 'linux';
    case 'windows': return 'windows';
    case 'ms windows': return 'windows';
    default: return false;
  }
};

export default platformIconClass;
