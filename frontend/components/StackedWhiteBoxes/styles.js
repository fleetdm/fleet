import styles from '../../styles';

const { color, font } = styles;

export default {
  boxStyles: {
    backgroundColor: color.white,
    borderRadius: '4px',
    boxShadow: '0 0 30px 0 rgba(0,0,0,0.30)',
    minHeight: '402px',
    boxSizing: 'border-box',
    padding: '30px',
    width: '524px',
    fontWeight: 300,
    position: 'absolute',
    top: '-399px',
    zIndex: '2',
  },
  containerStyles: {
    alignItems: 'center',
    display: 'flex',
    justifyContent: 'center',
    flexDirection: 'column',
  },
  exStyles: {
    color: color.lightGrey,
    textDecoration: 'none',
    position: 'absolute',
    top: '30px',
    right: '30px',
    fontWeight: 'bold',
  },
  exWrapperStyles: {
    textAlign: 'right',
    width: '100%',
  },
  headerStyles: {
    fontFamily: "'Oxygen', sans-serif",
    fontSize: font.large,
    fontWeight: '300',
    color: color.textUltradark,
    lineHeight: '32px',
    marginTop: 0,
    marginBottom: 0,
    textTransform: 'uppercase',
  },
  headerWrapperStyles: {
    width: '100%',
  },
  textStyles: {
    color: color.purpleGrey,
    fontSize: font.medium,
    lineHeight: '30px',
    letterSpacing: '0.64px',
  },
};
