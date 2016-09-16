import styles from '../../../styles';

const { border, padding } = styles;

export default {
  formStyles: {
    width: '100%',
  },
  inputStyles: {
    width: '100%',
  },
  submitButtonStyles: {
    borderTopLeftRadius: border.radius.base,
    borderTopRightRadius: border.radius.base,
    marginTop: padding.base,
    padding: padding.medium,
  },
};
