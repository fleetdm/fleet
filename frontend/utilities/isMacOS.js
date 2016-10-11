export default () => {
  const { navigator } = global.window;
  const macOSRegex = /(Mac|iPhone|iPod|iPad)/i;

  return macOSRegex.test(navigator.platform);
};
