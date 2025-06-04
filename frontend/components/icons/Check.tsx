import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ICheckProps {
  color?: Colors;
}

const Check = ({ color = "core-fleet-blue" }: ICheckProps) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      fill="none"
      viewBox="0 0 16 16"
      aria-label="check"
    >
      <path
        stroke={COLORS[color]}
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
        d="M3 10l3 3 7-10"
      />
    </svg>
  );
};

export default Check;
