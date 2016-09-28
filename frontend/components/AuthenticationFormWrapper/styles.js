import styles from '../../styles';

const { color } = styles;

export default {
  containerStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
    marginTop: '14vh',
  },
  whiteTabStyles: {
    backgroundColor: color.white,
    height: '30px',
    marginTop: '12px',
    borderTopLeftRadius: '4px',
    borderTopRightRadius: '4px',
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.3)',
    width: '376px',
  },
};
