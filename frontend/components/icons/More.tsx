import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IMoreProps {
  color?: Colors;
}

const More = ({ color = "core-fleet-blue" }: IMoreProps) => {
  return (
    <svg
      width="16"
      height="21"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 21"
    >
      <g fill={COLORS[color]}>
        <circle cx="8" cy="6" r="1" />
        <circle cx="8" cy="10" r="1" />
        <circle cx="8" cy="14" r="1" />
      </g>
    </svg>
  );
};

export default More;
