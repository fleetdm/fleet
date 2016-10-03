import styles from '../../../styles';

const { border, color, font, padding } = styles;
const baseStyles = {
  borderBottomLeftRadius: border.radius.base,
  borderBottomRightRadius: border.radius.base,
  borderTopLeftRadius: border.radius.base,
  borderTopRightRadius: border.radius.base,
  boxShadow: '0 3px 0 #734893',
  boxSizing: 'border-box',
  color: color.white,
  cursor: 'pointer',
  paddingBottom: padding.xSmall,
  paddingLeft: padding.xSmall,
  paddingRight: padding.xSmall,
  paddingTop: padding.xSmall,
  position: 'relative',
  textTransform: 'uppercase',
  ':active': {
    boxShadow: '0 1px 0 #734893, 0 -2px 0 #D1D9E9',
    top: '2px',
  },
  ':focus': {
    outline: 'none',
  },
};

export default {
  default: {
    ...baseStyles,
    backgroundColor: color.brandDark,
    borderBottom: `1px solid ${color.brandDark}`,
    borderLeft: `1px solid ${color.brandDark}`,
    borderRight: `1px solid ${color.brandDark}`,
    borderTop: `1px solid ${color.brandDark}`,
    boxShadow: `0 3px 0 ${color.brandLight}`,
    fontSize: font.medium,
  },
  inverse: {
    ...baseStyles,
    backgroundColor: color.white,
    borderBottom: `1px solid ${color.brandLight}`,
    borderLeft: `1px solid ${color.brandLight}`,
    borderRight: `1px solid ${color.brandLight}`,
    borderTop: `1px solid ${color.brandLight}`,
    boxShadow: `0 3px 0 ${color.brandDark}`,
    color: color.brandDark,
    fontSize: font.medium,
  },
  gradient: {
    ...baseStyles,
    backgroundImage: 'linear-gradient(134deg, #7166D9 0%, #C86DD7 100%)',
    backgroundColor: 'transparent',
    borderBottom: 'none',
    borderLeft: 'none',
    borderRight: 'none',
    borderTop: 'none',
    fontSize: font.large,
    fontWeight: '300',
    letterSpacing: '4px',
    paddingBottom: padding.medium,
    paddingLeft: padding.medium,
    paddingRight: padding.medium,
    paddingTop: padding.medium,
    width: '100%',
  },
};
