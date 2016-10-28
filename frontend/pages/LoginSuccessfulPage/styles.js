import styles from '../../styles';

const { color, font, padding } = styles;

export default {
  loginSuccessStyles: {
    color: color.success,
    textTransform: 'uppercase',
    fontSize: font.large,
    letterSpacing: '2px',
    fontWeight: '300',
  },
  subtextStyles: {
    fontSize: font.medium,
    color: color.accentText,
  },
  whiteBoxStyles: {
    backgroundColor: color.white,
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
    borderRadius: '4px',
    marginTop: '20px',
    paddingBottom: padding.base,
    paddingLeft: padding.base,
    paddingRight: padding.base,
    paddingTop: padding.most,
    textAlign: 'center',
    width: '340px',
    zIndex: '-1',
    alignSelf: 'center',
  },
};
