import Styles from '../../../../styles';

const { color, font, padding } = Styles;

export default (user, invite) => {
  const { admin, enabled } = user;
  let avatarFilter = 'none';
  const transition = 'all 0.3s ease-in-out';
  let userEmailTextColor = color.link;
  let userHeaderBgColor = '#F9F0FF';
  let userHeaderTextColor = color.textUltradark;
  let userStatusTextColor;
  let userWrapperBgColor = color.white;

  if (invite) {
    userStatusTextColor = color.brand;
  } else {
    if (admin) {
      userHeaderBgColor = color.brand;
      userHeaderTextColor = color.white;
    } else {
      userHeaderBgColor = color.white;
      userHeaderTextColor = color.textUltradark;
    }

    if (enabled) {
      userStatusTextColor = color.success;
    } else {
      userEmailTextColor = color.textMedium;
      avatarFilter = 'grayscale(100%)';
      userWrapperBgColor = color.bgMedium;
      userHeaderBgColor = color.textLight;
      userHeaderTextColor = color.textUltradark;
      userStatusTextColor = color.textMedium;
    }
  }

  const userHeaderStyles = {
    backgroundColor: userHeaderBgColor,
    borderBottom: `1px solid ${color.accentLight}`,
    color: userHeaderTextColor,
    height: '51px',
    marginBottom: padding.half,
    textAlign: 'center',
    transition,
    width: '100%',
  };

  return {
    avatarStyles: {
      border: `1px solid ${enabled ? color.brand : color.textMedium}`,
      filter: avatarFilter,
      display: 'block',
      marginLeft: 'auto',
      marginRight: 'auto',
      transition,
    },
    nameStyles: {
      lineHeight: '51px',
      margin: 0,
      padding: 0,
    },
    revokeInviteStyles: {
      position: 'absolute',
      width: '221px',
      bottom: '43px',
    },
    userDetailsStyles: {
      paddingLeft: padding.half,
      paddingRight: padding.half,
    },
    userEmailStyles: {
      fontSize: font.mini,
      color: userEmailTextColor,
      transition,
    },
    userHeaderStyles,
    userLabelStyles: {
      float: 'left',
      fontSize: font.small,
      fontWeight: admin ? 'bold' : 'normal',
      transition,
    },
    usernameStyles: {
      color: enabled ? color.brand : color.textMedium,
      fontSize: font.medium,
      textTransform: 'uppercase',
      transition,
    },
    userPositionStyles: {
      fontSize: font.small,
    },
    userStatusStyles: {
      color: userStatusTextColor,
      float: 'right',
      fontSize: font.small,
      textTransform: 'uppercase',
      transition,
    },
    userStatusWrapperStyles: {
      borderBottomColor: color.borderMedium,
      borderBottomStyle: 'solid',
      borderBottomWidth: '1px',
      borderTopColor: color.borderMedium,
      borderTopStyle: 'solid',
      borderTopWidth: '1px',
      marginTop: padding.half,
      paddingTop: padding.half,
      paddingBottom: padding.half,
    },
    userWrapperStyles: {
      backgroundColor: userWrapperBgColor,
      border: invite ? `1px dashed ${color.brand}` : 'none',
      boxShadow: '0 0 30px 0 rgba(0,0,0,0.30)',
      display: 'inline-block',
      height: '390px',
      marginLeft: padding.most,
      marginTop: padding.most,
      position: 'relative',
      transition,
      width: '239px',
    },
  };
};
