import styles from '../../../styles';

const { color, font, padding } = styles;
const FORM_WIDTH = '460px';

export default {
  containerStyles: {
    alignItems: 'center',
    backgroundColor: color.white,
    borderTopLeftRadius: '4px',
    borderTopRightRadius: '4px',
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
    marginTop: '-260px',
    opacity: 1,
    transform: 'scale(1.0)',
    width: '460px',
    alignSelf: 'center',
  },
  hideForm: {
    opacity: 0,
    transform: 'scale(1.3)',
    transition: 'all 300ms ease-in',
  },
};
