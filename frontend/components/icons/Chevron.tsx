import React from "react";

interface IChevronProps {
  color?: "coreVibrantBlue" | "coreFleetBlack";
  direction?: "up" | "down" | "left" | "right";
}

const SVG_PATH = {
  up:
    "M7.749 5.109 3.605 9.406a.385.385 0 0 0 0 .528l.927.957c.14.145.363.145.502 0L8 7.811l2.966 3.08c.14.145.363.145.502 0l.927-.957a.385.385 0 0 0 0-.528L8.251 5.11a.345.345 0 0 0-.502 0Z",
  down:
    "m8.751 10.891 4.144-4.297a.385.385 0 0 0 0-.528l-.927-.957a.345.345 0 0 0-.502 0L8.5 8.189l-2.966-3.08a.345.345 0 0 0-.502 0l-.927.957a.385.385 0 0 0 0 .528l4.144 4.297c.14.145.363.145.502 0Z",
  left:
    "m5.109 8.251 4.297 4.144c.145.14.383.14.528 0l.957-.927a.345.345 0 0 0 0-.502L7.811 8l3.08-2.966a.345.345 0 0 0 0-.502l-.957-.927a.385.385 0 0 0-.528 0L5.11 7.749a.345.345 0 0 0 0 .502Z",
  right:
    "M10.891 7.749 6.594 3.605a.385.385 0 0 0-.528 0l-.957.927a.345.345 0 0 0 0 .502L8.189 8l-3.08 2.966a.345.345 0 0 0 0 .502l.957.927c.145.14.383.14.528 0l4.297-4.144a.345.345 0 0 0 0-.502Z",
};

const FLEET_COLORS = {
  coreFleetBlack: "#6a67fe",
  coreVibrantBlue: "#6a67fe",
};

const Chevron = ({
  color = "coreFleetBlack",
  direction = "down",
}: IChevronProps) => {
  return (
    <svg width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d={SVG_PATH[direction]}
        fill={FLEET_COLORS[color]}
      />
    </svg>
  );
};

export default Chevron;
