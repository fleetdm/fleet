import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IRefresh {
  color?: Colors;
}
const Run = ({ color = "core-fleet-blue" }: IRefresh) => {
  return (
    <svg
      width="12"
      height="13"
      viewBox="0 0 12 13"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M1 11.5L1 1.5L11 6.83333L1 11.5Z"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinejoin="round"
      />
    </svg>
  );
};

export default Run;
