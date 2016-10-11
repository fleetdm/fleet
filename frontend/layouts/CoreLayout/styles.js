import Styles from '../../styles';

const { padding } = Styles;

export default {
  wrapperStyles: (showRightPanel) => {
    return {
      marginLeft: '241px',
      marginRight: showRightPanel ? '300px' : 0,
      paddingLeft: padding.base,
      paddingRight: padding.base,
      paddingTop: padding.base,
      '@media (max-width: 760px)': {
        marginLeft: '54px',
      },
    };
  },
};
