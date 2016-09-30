import Styles from '../../../../styles';

const { color, font, padding } = Styles;

const formSection = {
  borderTopColor: color.accentLight,
  borderTopStyle: 'solid',
  borderTopWidth: '1px',
  display: 'flex',
  justifyContent: 'space-between',
  marginTop: padding.base,
  paddingTop: padding.base,
};

const formInput = {
  fontSize: font.small,
  color: color.textDark,
  width: '300px',
};

const buttonStyles = {
  backgroundImage: 'none',
  backgroundColor: color.brandDark,
  boxShadow: '0 3px 0 #C38DEC',
  fontSize: font.medium,
  letterSpacing: '1px',
  paddingTop: padding.xSmall,
  paddingBottom: padding.xSmall,
  width: '200px',
};

export default {
  buttonStyles: {
    ...buttonStyles,
  },
  buttonInvertStyles: {
    ...buttonStyles,
    backgroundColor: color.white,
    borderColor: color.brandLight,
    borderStyle: 'solid',
    borderWidth: '1px',
    boxShadow: '0 3px 0 #9651CA',
    color: color.brandDark,
    marginRight: padding.half,
  },
  buttonWrapperStyles: {
    ...formSection,
    justifyContent: 'flex-end',
  },
  containerStyles: {},
  formSectionStyles: formSection,
  labelStyles: {
    display: 'block',
    marginBottom: padding.half,
  },
  moreOptionsIconStyles: {
    fontSize: font.xSmall,
    marginLeft: padding.half,
  },
  moreOptionsCtaSectionStyles: {
    ...formSection,
    borderTop: 'none',
    justifyContent: 'flex-end',
    paddingTop: 0,
  },
  moreOptionsTextStyles: {
    color: color.brand,
    cursor: 'pointer',
    fontSize: font.medium,
  },
  queryDescriptionInputStyles: {
    ...formInput,
  },
  queryHostsPercentageStyles: {
    ...formInput,
    display: 'inline-block',
    marginLeft: '5px',
    paddingRight: 0,
    maxWidth: '40px',
  },
  dropdownInputStyles: {
    ...formInput,
    backgroundColor: color.white,
    borderColor: color.brand,
    borderStyle: 'solid',
    borderWidth: '1px',
    height: '30px',
    textTransform: 'uppercase',
    ':focus': {
      outline: 'none',
    },
  },
  helpTextStyles: {
    backgroundColor: color.accentLight,
    padding: padding.base,
    width: '340px',
  },
  queryNameInputStyles: {
    ...formInput,
    height: '30px',
    lineHeight: '30px',
    paddingLeft: '3px',
  },
  queryNameWrapperStyles: {
    ...formSection,
  },
  runQuerySectionStyles: {
    ...formSection,
    display: 'block',
    textAlign: 'right',
  },
  runQueryTipStyles: {
    color: color.textLight,
    fontSize: font.small,
    marginRight: padding.half,
  },
};
