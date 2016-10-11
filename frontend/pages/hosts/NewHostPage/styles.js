import Styles from '../../../styles';

const { color, font, padding } = Styles;

export default {
  clipboardIconStyles: {
    position: 'absolute',
    right: '10px',
    cursor: 'pointer',
    top: '18px',
  },
  clipboardTextStyles: {
    color: color.brand,
    fontSize: font.xSmall,
    position: 'absolute',
    right: '10px',
    top: 0,
  },
  headerStyles: {
    borderBottom: `1px solid ${color.accentLight}`,
    fontSize: font.large,
    width: '364px',
  },
  hostTabHeaderStyles: (selected) => {
    const baseStyles = {
      backgroundColor: color.bgLight,
      color: color.textMedium,
      cursor: 'pointer',
      display: 'inline-block',
      fontSize: font.small,
      height: '36px',
      lineHeight: '36px',
      paddingLeft: padding.base,
      width: '235px',
    };
    const selectedStyles = {
      backgroundColor: color.bgMedium,
      color: color.brandLight,
    };

    if (!selected) return baseStyles;

    return {
      ...baseStyles,
      ...selectedStyles,
    };
  },
  inputStyles: {
    color: '#6f737f',
    borderRadius: '3px',
    backgroundColor: color.white,
    border: `solid 1px ${color.accentDark}`,
    boxShadow: 'inset 0 0 8px 0 rgba(0, 0, 0, 0.12)',
    boxSizing: 'border-box',
    fontFamily: 'SourceCodePro, Oxygen',
    fontSize: font.medium,
    height: '60px',
    letterSpacing: '1.2px',
    opacity: '0.8',
    paddingLeft: '28px',
    width: '100%',
  },
  scriptInfoWrapperStyles: {
    marginTop: padding.base,
  },
  selectedTabContentStyles: {
    backgroundColor: color.bgMedium,
    borderRadius: '3px',
    boxSizing: 'border-box',
    color: color.textUltradark,
    fontSize: font.small,
    height: '196px',
    paddingBottom: padding.base,
    paddingLeft: padding.base,
    paddingRight: padding.base,
    paddingTop: padding.base,
    width: '100%',
  },
  sectionWrapperStyles: {
    backgroundColor: color.white,
    marginBottom: padding.base,
    paddingBottom: padding.base,
    paddingLeft: padding.base,
    paddingRight: padding.base,
    paddingTop: padding.base,
  },
  textStyles: {
    color: color.TextUltradark,
    fontSize: font.small,
    lineHeight: font.large,
  },
};
