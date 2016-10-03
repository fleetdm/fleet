import Styles from '../../../../styles';

const { color, padding } = Styles;

export default {
  optionWrapperStyles: {
    color: color.textMedium,
    cursor: 'pointer',
    padding: padding.half,
    ':hover': {
      background: '#F9F0FF',
      color: color.textUltradark,
    },
  },
  selectWrapperStyles: {
    position: 'relative',
  },
};
