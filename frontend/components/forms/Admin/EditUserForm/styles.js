import Styles from '../../../../styles';

const { color, font, padding } = Styles;

export default {
  avatarWrapperStyles: {
    textAlign: 'center',
  },
  buttonWrapperStyles: {
    display: 'flex',
    flexDirection: 'row-reverse',
    justifyContent: 'space-between',
    marginTop: padding.half,
  },
  formButtonStyles: {
    paddingLeft: padding.base,
    paddingRight: padding.base,
  },
  formWrapperStyles: {
    boxSizing: 'border-box',
    paddingLeft: padding.half,
    paddingRight: padding.half,
  },
  inputStyles: {
    borderLeft: 'none',
    borderRight: 'none',
    borderTop: 'none',
    borderBottomWidth: '1px',
    fontSize: font.small,
    borderBottomStyle: 'solid',
    borderBottomColor: color.brand,
    color: color.textMedium,
    width: '100%',
  },
  inputWrapperStyles: {
    marginBottom: padding.half,
    marginTop: 0,
  },
  labelStyles: {
    color: color.textLight,
    textTransform: 'uppercase',
    fontSize: font.mini,
  },
};
