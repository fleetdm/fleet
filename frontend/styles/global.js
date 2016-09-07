import normalize from 'radium-normalize';
import { marginLonghand, paddingLonghand } from './helpers';
import color from './color';
import font from './font';
import padding from './padding';

const { auto, half, none, most } = padding;

const defaultTopAndBottomMargin = marginLonghand(most, ['Bottom', 'Top']);
const defaultLeftAndRightMargin = marginLonghand(auto, ['Left', 'Right']);
const defaultTopAndBottomPadding = paddingLonghand(none, ['Bottom', 'Top']);
const defaultLeftAndRightPadding = paddingLonghand(half, ['Left', 'Right']);
const MAX_WIDTH = '650px';

export default {
  ...normalize,
  body: {
    color: color.primary,
    ...defaultTopAndBottomMargin,
    ...defaultLeftAndRightMargin,
    ...defaultTopAndBottomPadding,
    ...defaultLeftAndRightPadding,
    fontSize: font.base,
    lineHeight: 1.6,
    maxWidth: MAX_WIDTH,
  },
  'h1, h2, h3': {
    lineHeight: 1.2,
  },
};
