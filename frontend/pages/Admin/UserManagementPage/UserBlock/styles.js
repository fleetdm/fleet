import Styles from '../../../../styles';

const { border, color, font, padding } = Styles;

export default {
  avatarStyles: {
    display: 'block',
    marginLeft: 'auto',
    marginRight: 'auto',
  },
  formButtonStyles: {
    paddingLeft: padding.medium,
    paddingRight: padding.medium,
  },
  nameStyles: {
    fontWeight: font.weight.bold,
    lineHeight: '51px',
    margin: 0,
    padding: 0,
  },
  userDetailsStyles: {
    paddingLeft: padding.half,
    paddingRight: padding.half,
  },
  userEmailStyles: {
    fontSize: font.mini,
    color: color.link,
  },
  userHeaderStyles: {
    backgroundColor: color.brand,
    color: color.white,
    height: '51px',
    marginBottom: padding.half,
    textAlign: 'center',
    width: '100%',
  },
  userLabelStyles: {
    float: 'left',
    fontSize: font.small,
  },
  usernameStyles: {
    color: color.brand,
    fontSize: font.medium,
    textTransform: 'uppercase',
  },
  userPositionStyles: {
    fontSize: font.small,
  },
  userStatusStyles: (enabled) => {
    return {
      color: enabled ? color.success : color.textMedium,
      float: 'right',
      fontSize: font.small,
    };
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
    boxShadow: border.shadow.blur,
    display: 'inline-block',
    height: '390px',
    marginLeft: padding.most,
    marginTop: padding.most,
    width: '239px',
  },
};
