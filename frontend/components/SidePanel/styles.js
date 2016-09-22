import Styles from '../../styles';

const { border, color, font, padding } = Styles;

const componentStyles = {
  companyLogoStyles: {
    position: 'absolute',
    left: '16px',
    height: '44px',
    marginRight: '10px',
    '@media (max-width: 760px)': {
      left: '4px',
    },
  },
  headerStyles: {
    borderBottomColor: color.accentLight,
    borderBottomStyle: 'solid',
    borderBottomWidth: '1px',
    height: '67px',
    marginBottom: padding.half,
    marginRight: padding.medium,
    paddingLeft: '54px',
  },
  iconStyles: {
    position: 'relative',
    fontSize: '22px',
    marginRight: '16px',
    top: '4px',
    left: 0,
    '@media (max-width: 760px)': {
      left: '5px',
    },
  },
  navItemBeforeStyles: {
    content: '',
    width: '6px',
    height: '50px',
    position: 'absolute',
    left: '-24px',
    top: '2px',
    bottom: 0,
    backgroundColor: '#9a61c6',
    '@media (max-width: 760px)': {
      left: 0,
    },
  },
  navItemListStyles: {
    listStyle: 'none',
    margin: 0,
    padding: 0,
  },
  navItemNameStyles: {
    display: 'inline-block',
    textDecoration: 'none',
    '@media (max-width: 760px)': {
      display: 'none',
    },
  },
  navItemStyles: (active) => {
    const activeStyles = {
      color: color.brand,
      borderBottom: 'none',
      transition: 'none',
      '@media (max-width: 760px)': {
        borderBottom: '8px solid #9a61c6',
        textAlign: 'center',
      },
    };

    const baseStyles = {
      minHeight: '40px',
      position: 'relative',
      color: color.textLight,
      cursor: 'pointer',
      fontSize: font.small,
      textTransform: 'uppercase',
      paddingTop: padding.half,
      transition: 'all 0.2s ease-in-out',
      '@media (max-width: 760px)': {
        textAlign: 'center',
      },
    };

    if (active) {
      return {
        ...baseStyles,
        ...activeStyles,
      };
    }

    return baseStyles;
  },
  navItemWrapperStyles: (lastChild) => {
    const baseStyles = {
      position: 'relative',
    };
    const lastChildStyles = {
      borderTopColor: color.accentLight,
      borderTopStyle: 'solid',
      borderTopWidth: '1px',
      marginRight: '16px',
      marginTop: '5px',
      '@media (max-width: 760px)': {
        marginRight: 0,
      },
    };

    if (lastChild) {
      return {
        ...baseStyles,
        ...lastChildStyles,
      };
    }

    return baseStyles;
  },
  navStyles: {
    backgroundColor: color.white,
    borderRightColor: color.borderMedium,
    borderRightStyle: 'solid',
    borderRightWidth: '1px',
    bottom: 0,
    boxShadow: '2px 0 8px 0 rgba(0, 0, 0, 0.1)',
    left: 0,
    paddingLeft: padding.large,
    paddingTop: padding.large,
    position: 'fixed',
    top: 0,
    width: '216px',
    '@media (max-width: 760px)': {
      paddingLeft: 0,
      width: '54px',
    },
  },
  orgNameStyles: {
    fontSize: font.medium,
    letterSpacing: '0.5px',
    margin: 0,
    overFlow: 'hidden',
    padding: 0,
    position: 'relative',
    textOverflow: 'ellipsis',
    top: '3px',
    whiteSpace: 'nowrap',
    '@media (max-width: 760px)': {
      display: 'none',
    },
  },
  subItemBeforeStyles: {
    backgroundColor: color.white,
    borderRadius: border.radius.circle,
    content: '',
    display: 'block',
    height: '7px',
    left: '24px',
    position: 'absolute',
    top: '15px',
    width: '7px',
  },
  subItemLinkStyles: (active) => {
    const activeStyles = {
      textDecoration: 'none',
      textTransform: 'none',
    };

    return active ? activeStyles : {};
  },
  subItemStyles: (active) => {
    const activeStyles = {
      fontWeight: font.weight.bold,
      opacity: '1',
    };

    const baseStyles = {
      color: color.white,
      marginBottom: '5px',
      marginLeft: 0,
      marginRight: 0,
      marginTop: '5px',
      opacity: '0.5',
      paddingBottom: padding.xSmall,
      paddingLeft: padding.most,
      paddingTop: padding.xSmall,
      position: 'relative',
      textTransform: 'none',
      transition: 'all 0.2s ease-in-out',
    };

    if (active) {
      return {
        ...baseStyles,
        ...activeStyles,
      };
    }

    return baseStyles;
  },
  subItemsStyles: (expanded) => {
    return {
      backgroundColor: color.brand,
      boxShadow: 'inset 0 5px 8px 0 rgba(0, 0, 0, 0.12), inset 0 -5px 8px 0 rgba(0, 0, 0, 0.12)',
      marginBottom: 0,
      marginRight: 0,
      minHeight: '87px',
      paddingBottom: padding.half,
      paddingTop: padding.half,
      marginLeft: '-24px',
      marginTop: padding.medium,
      transition: 'width 0.1s ease-in-out',
      '@media (max-width: 760px)': {
        bottom: '-8px',
        left: '54px',
        marginLeft: 0,
        position: 'absolute',
        width: expanded ? '251px' : '18px',
      },
    };
  },
  subItemListStyles: (expanded) => {
    return {
      listStyle: 'none',
      '@media (max-width: 760px)': {
        borderRight: '1px solid rgba(0,0,0,0.16)',
        display: expanded ? 'inline-block' : 'none',
        padding: 0,
        textAlign: 'left',
        width: '211px',
      },
    };
  },
  collapseSubItemsWrapper: {
    position: 'absolute',
    right: '3px',
    top: '41%',
    '@media (min-width: 761px)': {
      display: 'none',
    },
  },
  usernameStyles: {
    position: 'relative',
    top: '3px',
    display: 'inline-block',
    margin: 0,
    padding: 0,
    fontSize: font.small,
    textTransform: 'uppercase',
    '@media (max-width: 760px)': {
      display: 'none',
    },
  },
  userStatusStyles: (enabled) => {
    const backgroundColor = enabled ? color.success : color.warning;
    const size = '16px';

    return {
      backgroundColor,
      borderRadius: border.radius.circle,
      display: 'inline-block',
      height: size,
      left: '1px',
      marginRight: '6px',
      position: 'relative',
      top: '6px',
      width: size,
      '@media (max-width: 760px)': {
        display: 'none',
      },
    };
  },
};

export default componentStyles;
