import styles from '../../styles';

const { color, font, padding } = styles;

export default {
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
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
    borderRadius: '4px',
    marginBottom: '0px',
    marginLeft: 'auto',
    marginRight: 'auto',
    marginTop: padding.base,
    paddingBottom: padding.base,
    paddingLeft: padding.base,
    paddingRight: padding.base,
    paddingTop: padding.most,
    textAlign: 'center',
    width: '384px',
  },
};
