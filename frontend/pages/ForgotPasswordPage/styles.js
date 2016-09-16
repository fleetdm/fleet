import styles from '../../styles';

const { border, color, font, padding } = styles;

export default {
  emailSentButtonWrapperStyles: {
    backgroundColor: color.successLight,
    borderRadius: border.radius.base,
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
    borderRadius: border.radius.base,
    marginBottom: padding.base,
  },
  emailTextStyles: {
    color: color.link,
  },
};
