import React from "react";

import { COLORS, Colors } from "styles/var/colors";

interface IErrorProps {
  color?: Colors;
}

const Error = ({ color = "status-error" }: IErrorProps) => {
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
        d="M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8Zm0 3.25a.75.75 0 0 1 .75.75v5a.75.75 0 0 1-1.5 0V4A.75.75 0 0 1 8 3.25ZM8 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Error;
