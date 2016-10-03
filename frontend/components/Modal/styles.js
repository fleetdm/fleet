import styles from '../../styles';

const { color, padding } = styles;

export default {
  containerStyles: {
    position: 'fixed',
    top: 0,
    left: 0,
    bottom: 0,
    right: 0,
    backgroundColor: 'rgba(0,0,0,0.25)',
  },
  contentStyles: {
    paddingBottom: padding.half,
    paddingLeft: padding.half,
    paddingRight: padding.half,
    paddingTop: padding.half,
  },
  exStyles: {
    color: color.white,
    cursor: 'pointer',
    float: 'right',
    textDecoration: 'none',
    fontWeight: 'bold',
  },
  headerStyles: {
    backgroundColor: color.brand,
    color: color.white,
    paddingBottom: padding.half,
    paddingLeft: padding.half,
    paddingRight: padding.half,
    paddingTop: padding.half,
    textTransform: 'uppercase',
  },
  modalStyles: {
    backgroundColor: color.white,
    width: '400px',
    position: 'absolute',
    top: '30%',
    left: '30%',
  },
};
