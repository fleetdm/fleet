export const iconClassForLabel = (label) => {
  const lowerType = label.type && label.type.toLowerCase();
  const lowerDisplayText = label.display_text && label.display_text.toLowerCase();

  if (lowerType === 'all') return 'hosts';

  switch (lowerDisplayText) {
    case 'offline': return 'hosts';
    case 'online': return 'hosts';
    case 'mac os': return 'apple';
    case 'centos': return 'centos';
    case 'ubuntu': return 'ubuntu';
    case 'windows': return 'windows';
    default: return 'label';
  }
};

export default { iconClassForLabel };
