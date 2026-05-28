import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IArrowLeftProps {
  color?: Colors;
}

const ArrowLeft = ({ color = "ui-fleet-black-75" }: IArrowLeftProps) => {
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
        d="M6.707 3.707a1 1 0 0 0-1.414-1.414l-5 5a1 1 0 0 0 0 1.414l5 5a1 1 0 0 0 1.414-1.414L3.414 9H15a1 1 0 0 0 0-2H3.414l3.293-3.293Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ArrowLeft;
