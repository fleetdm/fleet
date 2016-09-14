import styles from '../../styles';

const { border, color, font, padding } = styles;

export default {
  containerStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
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
