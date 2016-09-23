import normalize from 'radium-normalize';
import { marginLonghand, paddingLonghand } from './helpers';
import color from './color';
import font from './font';
import padding from './padding';

const { none } = padding;
const defaultMargin = marginLonghand(none);
const defaultPadding = paddingLonghand(none);

export default (showBackgroundImage) => {
  const background = showBackgroundImage
    ? 'url("/assets/images/background.png") center center'
    : color.bgMedium;

  return {
    ...normalize,
    html: {
      position: 'relative',
      minHeight: '100%',
    },
    body: {
      background,
      backgroundSize: 'cover',
      color: color.textUltradark,
      ...defaultMargin,
      ...defaultPadding,
      fontFamily: 'Oxygen, sans-serif',
      fontSize: font.base,
      lineHeight: 1.6,
      margin: '0 0 94px',
    },
    'h1, h2, h3': {
      lineHeight: 1.2,
    },
  };
};
