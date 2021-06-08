const TOOLTIP_WIDTH = 300;
const calculateElementDistanceToBrowserRight = (el) => {
  const distanceWindowLeftToElementRight = el.getBoundingClientRect().right;
  const windowWidth = global.window.innerWidth;

  return windowWidth - distanceWindowLeftToElementRight;
};

export const calculateTooltipDirection = (el) => {
  const elementDistanceToBrowserRight = calculateElementDistanceToBrowserRight(
    el
  );

  return elementDistanceToBrowserRight < TOOLTIP_WIDTH ? "left" : "right";
};

export default { calculateTooltipDirection };
