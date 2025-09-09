import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IMoreProps {
  color?: Colors;
}

const More = ({ color = "ui-fleet-black-75" }: IMoreProps) => {
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
        d="M9 3a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm0 5a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm-1 6a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default More;
