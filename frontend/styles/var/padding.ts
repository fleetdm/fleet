import pxToRem from "./helpers";

export const PADDING = {
  "pad-auto": "auto",
  "pad-xxsmall": pxToRem(2),
  "pad-xsmall": pxToRem(4),
  "pad-small": pxToRem(8),
  "pad-icon": pxToRem(14),
  "pad-medium": pxToRem(16),
  "pad-large": pxToRem(24), // Vertical main page components gap
  "pad-xlarge": pxToRem(32), // Vertical page margins
  "pad-xxlarge": pxToRem(40),
  "pad-xxxlarge": pxToRem(80),
};

export type Padding = keyof typeof PADDING;
