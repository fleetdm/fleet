import styles from '../../../../styles';

const { color, font, padding } = styles;

export default {
  containerStyles: {
    marginTop: padding.base,
    position: 'relative',
    width: '100%',
  },
  errorStyles: {
    color: color.alert,
    fontSize: font.small,
    textTransform: 'lowercase',
  },
  iconStyles: (value) => {
    const baseStyles = {
      position: 'absolute',
      right: '6px',
      top: '28px',
      fontSize: '20px',
      color: color.accentText,
    };
    if (value) {
      return {
        ...baseStyles,
        color: color.brand,
      };
    }

    return baseStyles;
  },

  iconErrorStyles: (error) => {
    if (error) {
      return {
        color: color.alert,
      };
    }
    return false;
  },
  inputErrorStyles: (error) => {
    if (error) {
      return {
        borderBottomColor: color.alert,
      };
    }

    return {};
  },
  inputStyles: (value, type) => {
    const baseStyles = {
      borderLeft: 'none',
      borderRight: 'none',
      borderTop: 'none',
      borderBottomWidth: '2px',
      fontSize: '20px',
      borderBottomStyle: 'solid',
      borderBottomColor: color.brandUltralight,
      color: color.accentText,
      paddingRight: '30px',
      opacity: '1',
      textIndent: '1px',
      position: 'relative',
      width: '100%',
      boxSizing: 'border-box',
      ':focus': {
        outline: 'none',
      },
    };

    if (type === 'password' && value) {
      return {
        ...baseStyles,
        letterSpacing: '7px',
        color: color.textUltradark,
      };
    }

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
