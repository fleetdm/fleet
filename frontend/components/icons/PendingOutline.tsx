import React from "react";

import { COLORS, Colors } from "styles/var/colors";

interface ICheckProps {
  color?: Colors;
}

const PendingOutline = ({ color = "ui-fleet-black-50" }: ICheckProps) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      fill="none"
      viewBox="0 0 16 16"
    >
      <path
        fill={COLORS[color]}
        fillRule="evenodd"
        d="M14 8A6 6 0 112 8a6 6 0 0112 0zm2 0A8 8 0 110 8a8 8 0 0116 0zM5 9a1 1 0 100-2 1 1 0 000 2zm4-1a1 1 0 11-2 0 1 1 0 012 0zm2 1a1 1 0 100-2 1 1 0 000 2z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default PendingOutline;
