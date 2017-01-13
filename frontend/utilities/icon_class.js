export const statusIconClass = (status = '') => {
  const lowerStatus = status.toLowerCase();

  switch (lowerStatus) {
    case 'online':
      return 'success-check';
    case 'offline':
      return 'offline';
    case 'mia':
      return 'mia';
    default:
      return '';
  }
};

export const platformIconClass = (platform = '') => {
  if (!platform) return '';

  const lowerPlatform = platform.toLowerCase();

  switch (lowerPlatform) {
    case 'darwin':
      return 'apple';
    case 'linux':
      return 'penguin';
    default:
      return lowerPlatform;
  }
};

export default { platformIconClass, statusIconClass };
