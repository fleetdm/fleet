import Styles from '../../../styles';

const { color } = Styles;

export default {
  containerStyles: (saveQuery) => {
    return {
      backgroundColor: saveQuery ? color.brand : color.textMedium,
      borderTopLeftRadius: '12px',
      borderTopRightRadius: '12px',
      borderBottomRightRadius: '12px',
      borderBottomLeftRadius: '12px',
      border: '1px solid #eaeaea',
      cursor: 'pointer',
      display: 'inline-block',
      height: '22px',
      minWidth: '40px',
      position: 'relative',
      transition: 'backgroundColor 0.4s ease-in-out',
      width: '40px',
    };
  },
  buttonStyles: (saveQuery) => {
    return {
      marginTop: '3px',
      position: 'absolute',
      top: 0,
      transition: 'left 0.3s ease-in-out',
      left: saveQuery ? '21px' : '5px',
      height: '14px',
      width: '14px',
      borderRadius: '50%',
      backgroundColor: color.white,
    };
  },
};
