import Styles from '../../styles';

const { padding } = Styles;

export default {
  wrapperStyles: {
    marginLeft: '241px',
    paddingLeft: padding.base,
    paddingRight: padding.base,
    paddingTop: padding.base,
    '@media (max-width: 760px)': {
      marginLeft: '54px',
    },
  },
};
