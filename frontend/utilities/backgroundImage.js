/* eslint-disable no-mixed-operators */
const { document, window } = global;
const refreshDuration = 10000;
const SHAPE_DENSITY = 20;

let numPointsX;
let numPointsY;
let points;
let refreshTimeout;
let unitHeight;
let unitWidth;

const randomize = () => {
  const { length: pointsLength } = points;

  for (let i = 0; i < pointsLength; i++) {
    const { originX, originY } = points[i];

    if (originX !== 0 && originX !== (unitWidth * (numPointsX - 1))) {
      points[i].x = originX + Math.random() * unitWidth - unitWidth / 2;
    }

    if (originY !== 0 && originY !== (unitHeight * (numPointsY - 1))) {
      points[i].y = originY + Math.random() * unitHeight - unitHeight / 2;
    }
  }
};

const refresh = () => {
  randomize();

  const svgElement = document.querySelector('#bg svg');
  const childNodes = svgElement.childNodes;

  for (let i = 0; i < childNodes.length; i++) {
    const polygon = childNodes[i];
    const animate = polygon.childNodes[0];
    const point1 = points[polygon.point1];
    const point2 = points[polygon.point2];
    const point3 = points[polygon.point3];

    if (animate.getAttribute('to')) {
      animate.setAttribute('from', animate.getAttribute('to'));
    }

    animate.setAttribute('to', `${point1.x},${point1.y} ${point2.x},${point2.y} ${point3.x},${point3.y}`);
    animate.beginElement();
  }

  refreshTimeout = setTimeout(refresh, refreshDuration);
};

export const loadBackground = () => {
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  const appElement = document.querySelector('#bg');
  const { innerWidth } = window;
  const innerHeight = window.innerHeight - 20;
  const unitSize = (innerWidth + innerHeight) / SHAPE_DENSITY;
  svg.setAttribute('width', innerWidth);
  svg.setAttribute('height', innerHeight);
  appElement.appendChild(svg);

  numPointsX = Math.ceil(innerWidth / unitSize) + 1;
  numPointsY = Math.ceil(innerHeight / unitSize) + 1;
  unitWidth = Math.ceil(innerWidth / (numPointsX - 1));
  unitHeight = Math.ceil(innerHeight / (numPointsY - 1));

  points = [];

  for (let y = 0; y < numPointsY; y++) {
    for (let x = 0; x < numPointsX; x++) {
      const originX = unitWidth * x;
      const originY = unitHeight * y;

      points.push({
        x: originX,
        y: originY,
        originX,
        originY,
      });
    }
  }

  randomize();

  const { length: pointsLength } = points;

  for (let i = 0; i < pointsLength; i++) {
    const { originX, originY } = points[i];

    if (originX !== unitWidth * (numPointsX - 1) && originY !== unitHeight * (numPointsY - 1)) {
      const { x: topLeftX, y: topLeftY } = points[i];
      const { x: topRightX, y: topRightY } = points[i + 1];
      const { x: bottomLeftX, y: bottomLeftY } = points[i + numPointsX];
      const { x: bottomRightX, y: bottomRightY } = points[i + numPointsX + 1];

      const rando = Math.floor(Math.random() * 2);

      for (let n = 0; n < 2; n++) {
        const polygon = document.createElementNS(svg.namespaceURI, 'polygon');

        if (rando === 0) {
          if (n === 0) {
            polygon.point1 = i;
            polygon.point2 = i + numPointsX;
            polygon.point3 = i + numPointsX + 1;
            polygon.setAttribute('points', `${topLeftX},${topLeftY} ${bottomLeftX},${bottomLeftY} ${bottomRightX},${bottomRightY}`);
          } else if (n === 1) {
            polygon.point1 = i;
            polygon.point2 = i + 1;
            polygon.point3 = i + numPointsX + 1;
            polygon.setAttribute('points', `${topLeftX},${topLeftY} ${topRightX},${topRightY} ${bottomRightX},${bottomRightY}`);
          }
        } else if (rando === 1) {
          if (n === 0) {
            polygon.point1 = i;
            polygon.point2 = i + numPointsX;
            polygon.point3 = i + 1;
            polygon.setAttribute('points', `${topLeftX},${topLeftY} ${bottomLeftX},${bottomLeftY} ${topRightX},${topRightY}`);
          } else if (n === 1) {
            polygon.point1 = i + numPointsX;
            polygon.point2 = i + 1;
            polygon.point3 = i + numPointsX + 1;
            polygon.setAttribute('points', `${bottomLeftX},${bottomLeftY} ${topRightX},${topRightY} ${bottomRightX},${bottomRightY}`);
          }
        }

        polygon.setAttribute('fill', `rgba(0, 0, 0, ${Math.random() / 3})`);

        const animate = document.createElementNS('http://www.w3.org/2000/svg', 'animate');

        animate.setAttribute('fill', 'freeze');
        animate.setAttribute('attributeName', 'points');
        animate.setAttribute('dur', `${refreshDuration}ms`);
        animate.setAttribute('calcMode', 'linear');
        polygon.appendChild(animate);
        svg.appendChild(polygon);
      }
    }
  }

  refresh();
};

export const resizeBackground = () => {
  document.querySelector('#bg svg').remove();
  clearTimeout(refreshTimeout);
  loadBackground();
};

