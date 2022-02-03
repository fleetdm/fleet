export const scrollBy = (lines, pixelsPerLine) => {
  const { window } = global;

  window.scrollBy(0, -lines * pixelsPerLine);
};

export default scrollBy;
