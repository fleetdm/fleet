export const scrollToTop = () => {
  const { window } = global;

  const scrollStep = -window.scrollY / (500 / 15);
  const scrollInterval = setInterval(() => {
    if (window.scrollY !== 0) {
      window.scrollBy(0, scrollStep);
    } else {
      clearInterval(scrollInterval);
    }
  }, 15);
};

export default scrollToTop;
