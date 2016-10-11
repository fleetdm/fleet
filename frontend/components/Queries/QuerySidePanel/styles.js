import Styles from '../../../styles';

const { border, color, font, padding } = Styles;

export default {
  columnNameStyles: {
    paddingBottom: padding.xSmall,
    paddingLeft: padding.half,
    paddingRight: padding.half,
    paddingTop: padding.xSmall,
    backgroundColor: color.accentLight,
    border: `1px solid ${color.accentMedium}`,
    borderRadius: border.radius.base,
  },
  columnWrapperStyles: {
    alignItems: 'center',
    borderTop: `1px solid ${color.accentLight}`,
    color: color.textDark,
    display: 'flex',
    fontSize: font.small,
    justifyContent: 'space-between',
    paddingBottom: padding.half,
    paddingTop: padding.half,
  },
  containerStyles: {
    backgroundColor: color.white,
    borderLeftColor: color.borderMedium,
    borderLeftStyle: 'solid',
    borderLeftWidth: '1px',
    bottom: 0,
    boxShadow: '2px 0 8px 0 rgba(0, 0, 0, 0.1)',
    boxSizing: 'border-box',
    overflow: 'scroll',
    paddingBottom: '70px',
    paddingLeft: padding.small,
    paddingRight: padding.small,
    paddingTop: padding.small,
    position: 'fixed',
    right: 0,
    top: 0,
    width: '300px',
  },
  helpStyles: {
    marginLeft: padding.half,
    verticalAlign: 'middle',
  },
  loadSuggestedQueryStyles: {
    paddingBottom: '1px',
    paddingLeft: '5px',
    paddingRight: '5px',
    paddingTop: '1px',
  },
  numMoreColumnsStyles: {
    color: color.textMedium,
  },
  platformsTextStyles: {
    color: color.textMedium,
    fontSize: font.small,
    textTransform: 'capitalize',
  },
  sectionHeader: {
    fontSize: font.large,
    color: color.textMedium,
  },
  showAllColumnsStyles: {
    color: color.brand,
    cursor: 'pointer',
  },
  suggestedQueryStyles: {
    color: color.textMedium,
    fontSize: font.mini,
  },
  tableDescriptionStyles: {
    color: color.textMedium,
    fontSize: font.small,
  },
};
