export const statusIconClass = (status = '') => {
  const lowerStatus = status.toLowerCase();

  switch (lowerStatus) {
    case 'online':
      return 'kolidecon-success-check';
    case 'offline':
      return 'kolidecon-offline';
    default:
      return '';
  }
};

export const platformIconClass = (platform = '') => {
  const lowerPlatform = platform.toLowerCase();

  switch (lowerPlatform) {
    case 'darwin':
      return 'kolidecon-apple';
    default:
      return `kolidecon-${lowerPlatform}`;
  }
};

export default { platformIconClass, statusIconClass };
