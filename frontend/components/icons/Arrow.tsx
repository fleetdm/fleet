import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IArrowProps {
  color?: Colors;
}

const Arrow = ({ color = "ui-fleet-black-75" }: IArrowProps) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.293 3.707a1 1 0 0 1 1.414-1.414l5 5a1 1 0 0 1 0 1.414l-5 5a1 1 0 0 1-1.414-1.414L12.586 9H1a1 1 0 0 1 0-2h11.586L9.293 3.707Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Arrow;
