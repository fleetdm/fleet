import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ITransfer {
  color?: Colors;
}
const Transfer = ({ color = "core-fleet-blue" }: ITransfer) => {
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
        d="M5.707 1.707A1 1 0 0 0 4.293.293l-4 4a1 1 0 0 0 0 1.414l4 4a1 1 0 0 0 1.414-1.414L3.414 6H8a1 1 0 0 0 0-2H3.414l2.293-2.293ZM12.586 10l-2.293-2.293a1 1 0 1 1 1.414-1.414l4 4a1 1 0 0 1 0 1.414l-4 4a1 1 0 0 1-1.414-1.414L12.586 12H8a1 1 0 1 1 0-2h4.586Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Transfer;
