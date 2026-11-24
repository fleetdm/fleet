import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IErrorProps {
  color?: Colors;
  size?: IconSizes;
}

const Error = ({ color = "status-error", size = "medium" }: IErrorProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 10.48C8.552 10.48 9 10.928 9 11.48C9 12.032 8.552 12.48 8 12.48C7.447 12.48 7 12.032 7 11.48C7 10.928 7.447 10.48 8 10.48Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 3.5C8.414 3.5 8.75 3.835 8.75 4.25V8.75C8.75 9.164 8.414 9.5 8 9.5C7.585 9.5 7.25 9.164 7.25 8.75V4.25C7.25 3.835 7.585 3.5 8 3.5Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 0C12.418 0 16 3.581 16 8C16 12.418 12.418 16 8 16C3.581 16 0 12.418 0 8C0 3.581 3.581 0 8 0ZM8 1.5C4.41 1.5 1.5 4.41 1.5 8C1.5 11.589 4.41 14.5 8 14.5C11.589 14.5 14.5 11.589 14.5 8C14.5 4.41 11.589 1.5 8 1.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Error;
