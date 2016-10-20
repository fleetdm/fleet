import Styles from '../../../styles';

const { color, padding } = Styles;

export default {
  containerStyles: {
    backgroundColor: color.white,
    borderLeftColor: color.borderMedium,
    borderLeftStyle: 'solid',
    borderLeftWidth: '1px',
    bottom: 0,
    boxShadow: '2px 0 8px 0 rgba(0, 0, 0, 0.1)',
    boxSizing: 'border-box',
    overflow: 'scroll',
    paddingBottom: '70px',
    paddingLeft: padding.small,
    paddingRight: padding.small,
    paddingTop: padding.small,
    position: 'fixed',
    right: 0,
    top: 0,
    width: '300px',
  },
};
