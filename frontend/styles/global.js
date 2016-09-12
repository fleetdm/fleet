import normalize from 'radium-normalize';
import { marginLonghand, paddingLonghand } from './helpers';
import color from './color';
import font from './font';
import padding from './padding';

const { none } = padding;
const defaultMargin = marginLonghand(none);
const defaultPadding = paddingLonghand(none);

export default {
  ...normalize,
  body: {
    color: color.primary,
    ...defaultMargin,
    ...defaultPadding,
    display: 'flex',
    flexDirection: 'column',
    fontSize: font.base,
    lineHeight: 1.6,
    minHeight: '100vh',
  },
  'h1, h2, h3': {
    lineHeight: 1.2,
  },
  '#app': {
    minHeight: '100vh',
  },
  '#bg': {
    position: 'absolute',
    zIndex: '-1',
  },
};
