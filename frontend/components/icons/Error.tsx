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
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8zm1 13H7v-2h2v2zm-2-3h2V3H7v7z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Error;
