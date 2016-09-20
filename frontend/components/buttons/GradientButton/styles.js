import styles from '../../../styles';

const { border, color, font, padding } = styles;

export default {
  backgroundImage: 'linear-gradient(134deg, #7166D9 0%, #C86DD7 100%)',
  border: 'none',
  cursor: 'pointer',
  borderBottomLeftRadius: border.radius.base,
  borderBottomRightRadius: border.radius.base,
  borderTopLeftRadius: border.radius.base,
  borderTopRightRadius: border.radius.base,
  boxShadow: '0 3px 0 #734893',
  boxSizing: 'border-box',
  color: color.white,
  fontSize: font.large,
  fontWeight: '300',
  letterSpacing: '4px',
  marginTop: padding.base,
  padding: padding.medium,
  position: 'relative',
  textTransform: 'uppercase',
  width: '100%',
  ':active': {
    boxShadow: '0 1px 0 #734893, 0 -2px 0 #D1D9E9',
    top: '2px',
  },
  ':focus': {
    outline: 'none',
  },
};
