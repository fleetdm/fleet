import Style from '../../styles';

const { color } = Style;

export default {
  containerStyles: (alertType) => {
    const successAlert = {
      backgroundColor: color.success,
    };

    const baseStyles = {
      color: color.white,
    };

    if (alertType === 'success') {
      return {
        ...baseStyles,
        ...successAlert,
      };
    }

    return {};
  },
  contentStyles: {},
  undoStyles: {},
};
