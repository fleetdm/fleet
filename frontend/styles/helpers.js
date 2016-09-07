import { capitalize } from 'lodash';

const BASE_FONT_SIZE = 16;
const defaultSides = ['Bottom', 'Left', 'Right', 'Top'];

const calculateSpace = (type, amount, sides = defaultSides) => {
  const spaceObject = {};

  sides.forEach((side) => {
    spaceObject[`${type}${capitalize(side)}`] = amount;
  });

  return spaceObject;
};

export const marginLonghand = (amount, sides) => {
  return calculateSpace('margin', amount, sides);
};

export const paddingLonghand = (amount, sides) => {
  return calculateSpace('padding', amount, sides);
};

export const pxToRem = (targetSize) => {
  return `${targetSize / BASE_FONT_SIZE}rem`;
};
