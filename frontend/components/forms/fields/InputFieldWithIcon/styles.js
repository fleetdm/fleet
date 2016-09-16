import styles from '../../../../styles';

const { color, font, padding } = styles;

export default {
  containerStyles: {
    marginTop: padding.base,
    position: 'relative',
  },
  errorStyles: {
    color: color.alert,
    fontSize: font.small,
    textTransform: 'lowercase',
  },
  iconStyles: {
    position: 'absolute',
    right: '6px',
    top: '29px',
  },
  inputErrorStyles: (error) => {
    if (error) {
      return {
        borderBottomColor: color.alert,
      };
    }

    return {};
  },
  inputStyles: (value) => {
    const baseStyles = {
      borderLeft: 'none',
      borderRight: 'none',
      borderTop: 'none',
      borderBottomWidth: '1px',
      borderBottomStyle: 'solid',
      borderBottomColor: color.brand,
      color: color.accentText,
      width: '378px',
      ':focus': {
        outline: 'none',
      },
    };

    if (value) {
      return {
        ...baseStyles,
        color: color.textUltradark,
      };
    }

    return baseStyles;
  },
  placeholderStyles: (value) => {
    if (!value) return { visibility: 'hidden', height: '22px' };

    return {
      color: color.brand,
      fontSize: font.small,
      textTransform: 'lowercase',
    };
  },
};
