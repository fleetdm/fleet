const pxToRem = (px: number): string => {
  const baseSize = 16; // Assuming the base font size is 16px
  return `${px / baseSize}rem`;
};

export default pxToRem;
