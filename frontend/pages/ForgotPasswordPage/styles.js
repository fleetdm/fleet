import styles from '../../styles';

const { color, font, padding } = styles;

export default {
  emailSentButtonWrapperStyles: {
    backgroundColor: color.successLight,
    borderRadius: '4px',
    color: color.white,
    padding: padding.base,
    position: 'relative',
    textAlign: 'center',
    textTransform: 'uppercase',
  },
  emailSentIconStyles: {
    height: '35px',
    left: '18px',
    position: 'absolute',
    top: '14px',
    width: '35px',
  },
  emailSentTextStyles: {
    fontSize: font.medium,
  },
  emailSentTextWrapperStyles: {
    padding: padding.base,
    backgroundColor: color.accentLight,
    borderRadius: '4px',
    marginBottom: padding.base,
  },
  emailTextStyles: {
    color: color.link,
  },
};
