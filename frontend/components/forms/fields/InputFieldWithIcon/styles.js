import styles from '../../../../styles';

const { color, font, padding } = styles;

export default {
  containerStyles: {
    marginTop: padding.base,
    position: 'relative',
  },
  iconStyles: {
    position: 'absolute',
    right: '6px',
    top: '29px',
  },
  inputStyles: (value) => {
    const baseStyles = {
      borderLeft: 'none',
      borderRight: 'none',
      borderTop: 'none',
      borderBottomWidth: '1px',
      borderBottomStyle: 'solid',
      borderBottomColor: color.brightPurple,
      color: '#A2A1C8',
      width: '378px',
      ':focus': {
        outline: 'none',
      },
    };

    if (value) {
      return {
        ...baseStyles,
        color: color.grey,
      };
    }

    return baseStyles;
  },
  placeholderStyles: (value) => {
    if (!value) return { visibility: 'hidden', height: '22px' };

    return {
      color: color.brightPurple,
      fontSize: font.small,
      textTransform: 'lowercase',
    };
  },
};
