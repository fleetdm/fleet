import Styles from '../../../styles';

const { color, font, padding } = Styles;

export default {
  buttonWrapperStyles: {
    display: 'flex',
    flexDirection: 'row-reverse',
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
