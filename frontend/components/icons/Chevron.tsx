import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IChevronProps {
  color?: Colors;
  /** Default direction "down" */
  direction?: "up" | "down" | "left" | "right";
}

const SVG_PATH = {
  up: "M4 10L8 6L12 10",
  down: "m4 6 4 4 4-4",
  left: "M10 12L6 8L10 4",
  right: "M6 4L10 8L6 12",
};

const Chevron = ({
  color = "core-fleet-black",
  direction = "down",
}: IChevronProps) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        d={SVG_PATH[direction]}
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

export default Chevron;
