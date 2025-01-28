const pxToRem = (px: number): string => {
  const baseSize = 16; // Assuming the base font size is 16px
  return `${px / baseSize}rem`;
};

export const PADDING = {
  "pad-auto": "auto",
  "pad-xxsmall": pxToRem(2),
  "pad-xsmall": pxToRem(4),
  "pad-small": pxToRem(8),
  "pad-icon": pxToRem(14),
  "pad-medium": pxToRem(16),
  "pad-large": pxToRem(24),
  "pad-xlarge": pxToRem(32),
  "pad-xxlarge": pxToRem(40),
  "pad-xxxlarge": pxToRem(80),
};

export type Padding = keyof typeof PADDING;
