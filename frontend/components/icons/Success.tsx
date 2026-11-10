import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ISuccessProps {
  color?: Colors;
  size?: IconSizes;
}

const Success = ({
  color = "status-success",
  size = "medium",
}: ISuccessProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M10.61 5.203C10.873 4.883 11.346 4.837 11.666 5.1C11.985 5.363 12.031 5.836 11.768 6.156L7.708 11.086C7.574 11.25 7.377 11.348 7.166 11.358C6.954 11.368 6.749 11.289 6.599 11.139L4.279 8.82C3.986 8.527 3.986 8.052 4.279 7.759C4.572 7.466 5.046 7.466 5.339 7.759L7.075 9.494L10.61 5.203Z"
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

export default Success;
