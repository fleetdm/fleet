import Styles from '../../../styles';

const { color, font } = Styles;

export default {
  childrenWrapperStyles: (direction) => {
    return {
      position: 'absolute',
      top: '-22px',
      [direction]: '-218px',
      zIndex: 1,
    };
  },
  containerStyles: {
    display: 'inline-block',
    position: 'absolute',
    userSelect: 'none',
    MozUserSelect: 'none',
    WebkitUserSelect: 'none',
  },
  elipsisStyles: {
    color: color.textDark,
    fontSize: font.medium,
    fontWeight: 'bold',
    letterSpacing: '-1px',
  },
};
