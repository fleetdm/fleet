import styles from '../../styles';

const { color, padding } = styles;

export default {
  formWrapperStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
  },
  whiteTabStyles: {
    backgroundColor: color.white,
    height: '30px',
    marginTop: padding.base,
    borderTopLeftRadius: '4px',
    borderTopRightRadius: '4px',
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.3)',
    width: '400px',
  },
};
