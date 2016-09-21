import Styles from '../../styles';

const { border, color, font, padding } = Styles;

export default {
  companyLogoStyles: {
    position: 'absolute',
    left: '16px',
    height: '44px',
    marginRight: '10px',
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
  },
  navItemListStyles: {
    listStyle: 'none',
    margin: 0,
    padding: 0,
  },
  navItemNameStyles: {
    display: 'inline-block',
    textDecoration: 'none',
  },
  navItemStyles: (active) => {
    const activeStyles = {
      color: color.brand,
    };

    const baseStyles = {
      minHeight: '40px',
      position: 'relative',
      color: color.textLight,
      cursor: 'pointer',
      fontSize: font.small,
      textTransform: 'uppercase',
      paddingTop: padding.half,
      WebkitTransition: 'all 0.2s ease-in-out',
      MozTransition: 'all 0.2s ease-in-out',
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
      WebkitTransition: 'all 0.2s ease-in-out',
      MozTransition: 'all 0.2s ease-in-out',
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
  subItemsStyles: {
    backgroundColor: color.brand,
    boxShadow: 'inset 0 5px 8px 0 rgba(0, 0, 0, 0.12), inset 0 -5px 8px 0 rgba(0, 0, 0, 0.12)',
    listStyle: 'none',
    marginBottom: 0,
    marginLeft: '-24px',
    marginRight: 0,
    marginTop: padding.medium,
    minHeight: '6px',
    paddingBottom: padding.half,
    paddingTop: padding.half,
  },
  usernameStyles: {
    position: 'relative',
    top: '3px',
    display: 'inline-block',
    margin: 0,
    padding: 0,
    fontSize: font.small,
    textTransform: 'uppercase',
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
    };
  },
};
