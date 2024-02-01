import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ICheckProps {
  color?: Colors;
}

const SuccessOutline = ({ color = "status-success" }: ICheckProps) => {
  return (
    <svg width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="8" cy="8" r="7" stroke={COLORS[color]} strokeWidth="2" />
      <path
        d="m5 9 2 2 4-6"
        stroke={COLORS[color]}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

export default SuccessOutline;
