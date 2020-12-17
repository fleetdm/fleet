export const iconNameForLabel = (label) => {
  const lowerType = label.type && label.type.toLowerCase();
  const lowerDisplayText = label.display_text && label.display_text.toLowerCase();

  if (lowerType === 'all') return 'hosts-3';

  switch (lowerDisplayText || label) {
    case 'offline': return 'offline';
    case 'online': return 'online';
    case 'mia': return 'mia';
    case 'new': return 'new';
    case 'unknown': return 'hosts-2';
    default: return 'label';
  }
};

export const iconNameForPlatform = (platform = '') => {
  if (!platform.name && !platform) return false;

  const platformName = platform.name || platform;

  const lowerPlatform = platformName.toLowerCase();

  switch (lowerPlatform) {
    case 'macos': return 'apple-dark';
    case 'mac os x': return 'apple-dark';
    case 'mac osx': return 'apple-dark';
    case 'mac os': return 'apple-dark';
    case 'darwin': return 'apple-dark';
    case 'apple': return 'apple-dark';
    case 'centos': return 'centos-dark';
    case 'centos linux': return 'centos-dark';
    case 'ubuntu': return 'ubuntu-dark';
    case 'ubuntu linux': return 'ubuntu-dark';
    case 'linux': return 'linux-dark';
    case 'windows': return 'windows-dark';
    case 'ms windows': return 'windows-dark';
    default: return false;
  }
};
