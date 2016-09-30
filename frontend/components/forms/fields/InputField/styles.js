import styles from '../../../../styles';

const { color, font } = styles;

export default {
  componentLabelStyles: (error) => {
    if (!error) return {};

    return {
      color: color.alert,
    };
  },
  inputErrorStyles: (error) => {
    if (!error) return {};

    return {
      borderColor: color.alert,
    };
  },
  inputStyles: (type, value) => {
    const baseStyles = {
      borderColor: color.brand,
      borderRadius: '2px',
      borderStyle: 'solid',
      borderWidth: '1px',
      fontSize: font.base,
      color: color.accentText,
      paddingRight: '30px',
      opacity: '1',
      textIndent: '1px',
      position: 'relative',
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
};
