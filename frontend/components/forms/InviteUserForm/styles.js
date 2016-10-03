import Styles from '../../../styles';

const { color, font, padding } = Styles;

export default {
  buttonStyles: {
    fontSize: font.small,
    height: '38px',
    marginButtom: '5px',
    paddingBottom: 0,
    paddingLeft: 0,
    paddingRight: 0,
    paddingTop: 0,
    width: '180px',
  },
  buttonWrapperStyles: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  radioElementStyles: {
    paddingBottom: padding.base,
  },
  roleTitleStyles: {
    color: color.brand,
    fontSize: font.mini,
    marginBottom: 0,
  },
};
