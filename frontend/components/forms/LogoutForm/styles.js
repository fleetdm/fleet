import styles from '../../../styles';

const { border, color, font, padding } = styles;
const FORM_WIDTH = '460px';

export default {
  avatarStyles: {
    borderWidth: '1px',
    borderStyle: 'solid',
    borderColor: color.brand,
    borderRadius: '50%',
  },
  containerStyles: {
    alignItems: 'center',
    backgroundColor: color.white,
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    boxSizing: 'border-box',
    display: 'flex',
    flexDirection: 'column',
    padding: '30px',
    width: FORM_WIDTH,
    minHeight: '350px',
  },
  formStyles: {
    boxShadow: '0 5px 30px 0 rgba(0,0,0,0.30)',
  },
  submitButtonStyles: {
    width: FORM_WIDTH,
  },
  subtextStyles: {
    color: color.textLight,
    fontSize: font.medium,
    marginTop: padding.half,
  },
  usernameStyles: {
    color: color.brand,
    fontSize: font.large,
    marginBottom: padding.half,
    marginTop: padding.half,
    textTransform: 'uppercase',
  },
};
