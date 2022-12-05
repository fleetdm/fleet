import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IExProps {
  color?: Colors;
}

const Ex = ({ color = "status-error" }: IExProps) => {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="m3 3 10 10M3 13 13 3"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
};

export default Ex;
