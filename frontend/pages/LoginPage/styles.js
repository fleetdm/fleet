import styles from '../../styles';

const { color, padding } = styles;

export default {
  containerStyles: {
    alignItems: 'center',
    display: 'flex',
    flexDirection: 'column',
    paddingTop: '10%',
  },
  formWrapperStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
  },
  whiteTabStyles: {
    backgroundColor: color.white,
    height: '20px',
    marginTop: padding.base,
    width: '384px',
  },
};
