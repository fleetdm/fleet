import styles from '../../styles';

const { border, color, font, padding } = styles;

export default {
  boxStyles: {
    backgroundColor: color.white,
    borderRadius: border.radius.base,
    boxShadow: border.shadow.blur,
    minHeight: '370px',
    boxSizing: 'border-box',
    padding: '30px',
    width: '524px',
    position: 'relative',
    fontWeight: 300,
  },
  containerStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
  },
  exStyles: {
    color: color.lightGrey,
    textDecoration: 'none',
    position: 'absolute',
    top: '30px',
    right: '30px',
    fontWeight: 'bold',
  },
  exWrapperStyles: {
    textAlign: 'right',
    width: '100%',
  },
  headerStyles: {
    fontFamily: "'Oxygen', sans-serif",
    fontSize: font.large,
    fontWeight: '300',
    color: color.textUltradark,
    lineHeight: '32px',
    marginTop: 0,
    marginBottom: 0,
    textTransform: 'uppercase',
  },
  headerWrapperStyles: {
    width: '100%',
  },
  tabStyles: {
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxShadow: border.shadow.blur,
    height: '20px',
    width: '460px',
  },
  textStyles: {
    color: color.purpleGrey,
    fontSize: font.medium,
    lineHeight: '30px',
    letterSpacing: '0.64px',
  },
  smallTabStyles: {
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxShadow: border.shadow.blur,
    height: '20px',
    marginTop: padding.base,
    width: '400px',
  },
};
