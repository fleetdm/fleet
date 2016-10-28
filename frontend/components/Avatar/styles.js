export default (size) => {
  const lowercaseSize = size && size.toLowerCase();
  const baseStyles = {
    borderRadius: '50%',
  };
  const smallStyles = {
    height: '20px',
    width: '20px',
  };

  if (lowercaseSize === 'small') {
    return {
      ...baseStyles,
      ...smallStyles,
    };
  }

  return baseStyles;
};
