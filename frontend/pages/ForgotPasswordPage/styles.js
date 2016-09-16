import styles from '../../styles';

const { border, color, font, padding } = styles;

export default {
  containerStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
  },
  emailSentButtonWrapperStyles: {
    backgroundColor: color.successLight,
    borderRadius: border.radius.base,
    color: color.white,
    padding: padding.base,
    position: 'relative',
    textAlign: 'center',
    textTransform: 'uppercase',
  },
  emailSentIconStyles: {
    height: '35px',
    left: '18px',
    position: 'absolute',
    top: '14px',
    width: '35px',
  },
  emailSentTextStyles: {
    fontSize: font.medium,
  },
  emailSentTextWrapperStyles: {
    padding: padding.base,
    backgroundColor: color.accentLight,
    borderRadius: border.radius.base,
    marginBottom: padding.base,
  },
  emailTextStyles: {
    color: color.link,
  },
  forgotPasswordStyles: {
    alignItems: 'center',
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxShadow: border.shadow.blur,
    boxSizing: 'border-box',
    display: 'flex',
    flexDirection: 'column',
    padding: padding.base,
    width: '522px',
  },
  headerStyles: {
    fontFamily: "'Oxygen', sans-serif",
    fontSize: font.large,
    fontWeight: '300',
    color: color.mediumGrey,
    lineHeight: '32px',
    textTransform: 'uppercase',
  },
  smallWhiteTabStyles: {
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxShadow: border.shadow.blur,
    height: '20px',
    marginTop: padding.base,
    width: '400px',
  },
  textStyles: {
    color: color.purpleGrey,
    fontSize: font.medium,
  },
  whiteTabStyles: {
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxShadow: border.shadow.blur,
    height: '20px',
    width: '460px',
  },
};
