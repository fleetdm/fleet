import styles from '../../../styles';

const { border, color, font, padding } = styles;

export default (disabled) => {
  const cursor = disabled ? 'not-allowed' : 'pointer';

  return {
    backgroundImage: 'linear-gradient(134deg, #7166D9 0%, #C86DD7 100%)',
    border: 'none',
    cursor,
    borderBottomLeftRadius: border.radius.base,
    borderBottomRightRadius: border.radius.base,
    boxSizing: 'border-box',
    color: color.white,
    fontSize: font.large,
    letterSpacing: '4px',
    padding: padding.base,
    fontWeight: '300',
    textTransform: 'uppercase',
    width: '100%',
    boxShadow: '0 3px 0 #734893',
    position: 'relative',
    ':active': {
      top: '2px',
      boxShadow: '0 1px 0 #734893, 0 -2px 0 #D1D9E9',
    },
    ':focus': {
      outline: 'none',
    },
  };
};
