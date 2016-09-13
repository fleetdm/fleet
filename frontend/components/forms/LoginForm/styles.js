import styles from '../../../styles';

const { border, color, font, padding } = styles;
const FORM_WIDTH = '460px';

export default {
  containerStyles: {
    alignItems: 'center',
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxSizing: 'border-box',
    display: 'flex',
    flexDirection: 'column',
    padding: padding.base,
    width: FORM_WIDTH,
    minHeight: '350px',
  },
  formStyles: {
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
  },
  submitButtonStyles: (canSubmit) => {
    const cursor = canSubmit ? 'pointer' : 'not-allowed';

    return {
      backgroundImage: 'linear-gradient(134deg, #7166D9 0%, #C86DD7 100%)',
      border: 'none',
      cursor,
      borderBottomLeftRadius: border.radius.base,
      borderBottomRightRadius: border.radius.base,
      boxSizing: 'border-box',
      color: color.white,
      fontSize: font.large,
      letterSpacing: '4px',
      padding: padding.base,
      fontWeight: '300',
      textTransform: 'uppercase',
      width: FORM_WIDTH,
      boxShadow: '0 3px 0 #734893',
      position: 'relative',
      ':active': {
        top: '2px',
        boxShadow: '0 1px 0 #734893, 0 -2px 0 #D1D9E9',
      },
      ':focus': {
        outline: 'none',
      },
    };
  },
};
