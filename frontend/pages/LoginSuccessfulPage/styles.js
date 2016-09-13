import styles from '../../styles';

const { color, font, padding } = styles;

export default {
  containerStyles: {
    paddingTop: '100px',
    textAlign: 'center',
  },
  loginSuccessStyles: {
    color: color.green,
    textTransform: 'uppercase',
    fontSize: font.large,
    letterSpacing: '2px',
    fontWeight: '300',
  },
  subtextStyles: {
    fontSize: font.medium,
    color: color.lightGrey,
  },
  whiteBoxStyles: {
    backgroundColor: color.white,
    margin: '0 auto',
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
    borderRadius: '4px',
    marginTop: padding.base,
    padding: padding.base,
    paddingTop: padding.most,
    textAlign: 'center',
    width: '384px',
  },
};
