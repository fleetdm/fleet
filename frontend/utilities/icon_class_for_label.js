export const iconClassForLabel = (label) => {
  const lowerType = label.type && label.type.toLowerCase();
  const lowerDisplayText = label.display_text && label.display_text.toLowerCase();

  if (lowerType === 'all') return 'hosts';

  switch (lowerDisplayText || label) {
    case 'offline': return 'offline';
    case 'online': return 'success-check';
    case 'mia': return 'mia';
    case 'unknown': return 'single-host';
    default: return 'label';
  }
};

export default iconClassForLabel;
