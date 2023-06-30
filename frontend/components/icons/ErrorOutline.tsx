import React from "react";

import { COLORS, Colors } from "styles/var/colors";

interface IErrorOutlineProps {
  color?: Colors;
}

const ErrorOutline = ({ color = "status-error" }: IErrorOutlineProps) => {
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
        d="M8 14A6 6 0 1 0 8 2a6 6 0 0 0 0 12Zm0 2A8 8 0 1 0 8 0a8 8 0 0 0 0 16ZM8 4a1 1 0 0 1 1 1v3a1 1 0 1 1-2 0V5a1 1 0 0 1 1-1Zm0 8a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ErrorOutline;
