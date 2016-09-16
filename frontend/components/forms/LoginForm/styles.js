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
  forgotPasswordStyles: {
    fontSize: font.medium,
    textDecoration: 'none',
    color: color.accentText,
  },
  forgotPasswordWrapperStyles: {
    marginTop: padding.base,
    textAlign: 'right',
    width: '378px',
  },
  formStyles: {
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
  },
  submitButtonStyles: {
    width: FORM_WIDTH,
  },
};
