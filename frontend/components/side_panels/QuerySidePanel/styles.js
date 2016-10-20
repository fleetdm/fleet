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
