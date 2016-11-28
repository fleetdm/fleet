export const statusIconClass = (status = '') => {
  const lowerStatus = status.toLowerCase();

  switch (lowerStatus) {
    case 'online':
      return 'success-check';
    case 'offline':
      return 'offline';
    default:
      return '';
  }
};

export const platformIconClass = (platform = '') => {
  const lowerPlatform = platform.toLowerCase();

  switch (lowerPlatform) {
    case 'darwin':
      return 'apple';
    default:
      return lowerPlatform;
  }
};

export default { platformIconClass, statusIconClass };
