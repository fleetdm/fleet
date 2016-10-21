import Styles from '../../../styles';

const { color, font, padding } = Styles;

export default {
  containerStyles: (status) => {
    const baseStyles = {
      backgroundColor: color.white,
      borderBottom: `solid 1px ${color.silver}`,
      borderLeft: `solid 1px ${color.silver}`,
      borderRight: `solid 1px ${color.silver}`,
      borderTop: `solid 1px ${color.silver}`,
      borderRadius: '3px',
      boxShadow: '0 2px 8px 0 rgba(0, 0, 0, 0.05)',
      boxSizing: 'border-box',
      display: 'inline-block',
      height: '286px',
      marginLeft: padding.base,
      marginTop: padding.base,
      paddingBottom: padding.half,
      paddingLeft: padding.half,
      paddingRight: padding.half,
      paddingTop: padding.half,
      position: 'relative',
      textAlign: 'center',
      width: '240px',
    };
    const statusStyles = {
      online: {
        borderTop: `6px solid ${color.success}`,
      },
      offline: {
        borderTop: `6px solid ${color.alert}`,
      },
    };

    return {
      ...baseStyles,
      ...statusStyles[status],
    };
  },
  contentSeparatorStyles: {
    borderTop: `1px solid ${color.accentLight}`,
    marginTop: padding.half,
  },
  disableIconStyles: {
    fontSize: font.larger,
  },
  elipsisChildItemStyles: {
    color: color.textMedium,
    width: '60px',
  },
  elipsisChidrenWrapperStyles: {
    alignItems: 'center',
    backgroundColor: color.white,
    border: `1px solid ${color.textMedium}`,
    borderRadius: '3px',
    boxShadow: '0 2px 8px 0 rgba(0, 0, 0, 0.05)',
    boxSizing: 'border-box',
    display: 'flex',
    height: '100px',
    justifyContent: 'space-around',
    width: '200px',
  },
  elipsisPositionStyles: {
    top: '-3px',
    right: '10px',
  },
  hostContentItemStyles: {
    color: color.textUltradark,
    fontSize: font.small,
  },
  hostnameStyles: {
    color: color.link,
    fontSize: font.mini,
    fontWeight: 'bold',
    marginTop: 0,
    marginBottom: 0,
  },
  iconStyles: {
    color: color.textDark,
    fontSize: font.mini,
    marginRight: '3px',
  },
  monoStyles: {
    fontFamily: 'SourceCodePro',
  },
  queryIconStyles: {
    color: color.brand,
    fontSize: font.larger,
  },
  statusStyles: (status) => {
    const baseStyles = {
      fontSize: font.medium,
      textAlign: 'left',
      textTransform: 'uppercase',
    };
    const statusStyles = {
      online: {
        color: color.success,
      },
      offline: {
        color: color.alert,
      },
    };

    return {
      ...baseStyles,
      ...statusStyles[status],
    };
  },
  verticleRuleStyles: {
    borderRight: `1px dashed ${color.textLight}`,
    height: '62%',
    width: '1px',
  },
};
