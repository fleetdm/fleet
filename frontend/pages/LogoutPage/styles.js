import styles from '../../styles';

const { color } = styles;

export default {
  whiteTabStyles: {
    backgroundColor: color.white,
    height: '30px',
    marginTop: '20px',
    borderTopLeftRadius: '4px',
    borderTopRightRadius: '4px',
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.3)',
    width: '376px',
    alignSelf: 'center',
  },
  authWrapperStyles: {
    alignItems: 'center',
    display: 'flex',
    flexDirection: 'column',
    position: 'relative',
    height: 'calc(100vh - 94px)',
    justifyContent: 'center',
  },
};
