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
    padding: '30px',
    width: FORM_WIDTH,
    minHeight: '350px',
    fontWeight: '300',
  },
  forgotPasswordStyles: {
    fontSize: font.medium,
    letterSpacing: '1px',
    textDecoration: 'none',
    color: color.accentText,
  },
  forgotPasswordWrapperStyles: {
    marginTop: padding.base,
    textAlign: 'right',
    width: '100%',
  },
  formStyles: {
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
  },
  submitButtonStyles: {
    borderTopLeftRadius: 0,
    borderTopRightRadius: 0,
    paddingBottom: padding.base,
    paddingLeft: padding.base,
    paddingRight: padding.base,
    paddingTop: padding.base,
    width: FORM_WIDTH,
  },
};
