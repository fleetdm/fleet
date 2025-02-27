import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IPlusProps {
  color?: Colors;
}

const Plus = ({ color = "core-fleet-blue" }: IPlusProps) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      aria-label="plus"
    >
      <path
        d="M8 3v10M3 8h10"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
};

export default Plus;
