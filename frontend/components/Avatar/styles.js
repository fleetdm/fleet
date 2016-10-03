import Styles from '../../styles';

const { border } = Styles;

export default (size) => {
  const lowercaseSize = size && size.toLowerCase();
  const baseStyles = {
    borderRadius: border.radius.circle,
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
