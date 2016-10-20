import Styles from '../../../styles';

const { border, color, font, padding } = Styles;

const componentStyles = {
  companyLogoStyles: {
    position: 'absolute',
    left: '0',
    top: '23px',
    height: '42px',
    marginRight: '10px',
    borderColor: color.accentMedium,
    borderStyle: 'solid',
    borderWidth: '1px',
    borderRadius: '100%',
    '@media (max-width: 760px)': {
      left: '5px',
    },
  },
  headerStyles: {
    borderBottomColor: color.accentLight,
    borderBottomStyle: 'solid',
    borderBottomWidth: '1px',
    height: '62px',
    cursor: 'pointer',
    paddingLeft: '54px',
    paddingTop: '26px',
    marginRight: '16px',
    position: 'relative',
  },
  orgChevronStyles: {
    color: color.accentMedium,
    fontSize: '12px',
    position: 'absolute',
    top: '50px',
    right: '35px',
    '@media (max-width: 760px)': {
      top: 'auto',
      left: '0',
      right: '0',
      bottom: '6px',
      textAlign: 'center',
      display: 'block',
    },
  },
  iconStyles: {
    position: 'relative',
    fontSize: '22px',
    marginRight: '16px',
    top: '4px',
    left: 0,
    '@media (max-width: 760px)': {
      display: 'block',
      textAlign: 'center',
      marginRight: 0,
    },
  },
  navItemBeforeStyles: {
    content: '',
    width: '6px',
    height: '50px',
    position: 'absolute',
    left: '-16px',
    top: 0,
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
      ':hover': {
        color: color.brandDark,
      },
      '@media (max-width: 760px)': {
        borderBottom: '6px solid #9a61c6',
      },
    };

    const baseStyles = {
      minHeight: '40px',
      position: 'relative',
      color: color.textLight,
      cursor: 'pointer',
      fontSize: '13px',
      letterSpacing: '0.5px',
      textTransform: 'uppercase',
      paddingTop: padding.half,
      transition: 'color 0.2s ease-in-out',
      ':hover': {
        color: color.textDark,
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
    boxSizing: 'border-box',
    left: 0,
    paddingLeft: '16px',
    position: 'fixed',
    top: 0,
    width: '223px',
    '@media (max-width: 760px)': {
      paddingLeft: 0,
      width: '54px',
    },
  },
  orgNameStyles: {
    fontSize: '16px',
    letterSpacing: '0.5px',
    margin: 0,
    overFlow: 'hidden',
    padding: 0,
    position: 'relative',
    textOverflow: 'ellipsis',
    top: '1px',
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
      fontSize: '13px',
      fontWeight: font.weight.bold,
      opacity: '1',
      ':hover': {
        opacity: '1.0',
      },
    };

    const baseStyles = {
      color: color.white,
      cursor: 'pointer',
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
      ':hover': {
        opacity: '0.75',
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
  subItemsStyles: {
    backgroundColor: color.brand,
    boxShadow: 'inset 0 5px 8px 0 rgba(0, 0, 0, 0.12), inset 0 -5px 8px 0 rgba(0, 0, 0, 0.12)',
    fontSize: '13px',
    marginRight: 0,
    marginBottom: '6px',
    paddingBottom: '3px',
    paddingTop: '3px',
    marginLeft: '-16px',
    position: 'relative',
    top: '0px',
    transition: 'width 0.1s ease-in-out',
    '@media (max-width: 760px)': {
      minHeight: '84px',
      borderTopRightRadius: '3px',
      borderBottomRightRadius: '3px',
      boxShadow: '2px 2px 8px rgba(0,0,0,0.1)',
      bottom: '-8px',
      left: '54px',
      marginLeft: 0,
      position: 'absolute',
    },
  },
  subItemListStyles: (expanded) => {
    return {
      listStyle: 'none',
      paddingLeft: '16px',
      minHeight: '87px',
      '@media (max-width: 760px)': {
        borderRight: '1px solid rgba(0,0,0,0.16)',
        display: expanded ? 'inline-block' : 'none',
        padding: 0,
        textAlign: 'left',
        width: '166px',
      },
    };
  },
  collapseSubItemsWrapper: {
    bottom: '0',
    color: color.white,
    cursor: 'pointer',
    lineHeight: '95px',
    position: 'absolute',
    right: '4px',
    top: '0',
    '@media (min-width: 761px)': {
      display: 'none',
    },
  },
  usernameStyles: {
    position: 'relative',
    display: 'inline-block',
    margin: 0,
    padding: 0,
    top: '-3px',
    left: '4px',
    fontSize: '13px',
    letterSpacing: '0.6px',
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
      marginRight: '6px',
      position: 'relative',
      width: size,
      '@media (max-width: 760px)': {
        display: 'none',
      },
    };
  },
};

export default componentStyles;
