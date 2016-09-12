
import styles from '../../../styles';

const { border, color, font, padding } = styles;
const FORM_WIDTH = '480px';

export default {
  containerStyles: {
    alignItems: 'center',
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxShadow: '0 0 30px 0 rgba(0,0,0,0.30)',
    boxSizing: 'border-box',
    display: 'flex',
    flexDirection: 'column',
    padding: padding.base,
    width: FORM_WIDTH,
  },
  submitButtonStyles: (canSubmit) => {
    const bgColor = {
      start: canSubmit ? '#7166D9' : '#B2B2B2',
      end: canSubmit ? '#C86DD7' : '#C7B7C9',
    };

    return {
      backgroundImage: `linear-gradient(to bottom right, ${bgColor.start}, ${bgColor.end})`,
      border: 'none',
      borderBottomLeftRadius: border.radius.base,
      borderBottomRightRadius: border.radius.base,
      boxSizing: 'border-box',
      color: color.white,
      cursor: canSubmit ? 'pointer' : 'not-allowed',
      fontSize: font.large,
      letterSpacing: '4px',
      padding: padding.base,
      textTransform: 'uppercase',
      width: FORM_WIDTH,
      ':focus': {
        outline: 'none',
      },
    };
  },
  userIconStyles: {
  },
};
