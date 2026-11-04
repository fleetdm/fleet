import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICheckProps {
  color?: Colors;
  size?: IconSizes;
}

const PendingOutline = ({
  color = "ui-fleet-black-50",
  size = "medium",
}: ICheckProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M4.635 7C5.188 7 5.635 7.447 5.635 8C5.635 8.552 5.188 9 4.635 9C4.083 9 3.635 8.552 3.635 8C3.635 7.447 4.083 7 4.635 7Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 7C8.552 7 9 7.447 9 8C9 8.552 8.552 9 8 9C7.447 9 7 8.552 7 8C7 7.447 7.447 7 8 7Z"
        fill={COLORS[color]}
      />
      <path
        d="M11.363 7C11.915 7 12.363 7.447 12.363 8C12.363 8.552 11.915 9 11.363 9C10.811 9 10.363 8.552 10.363 8C10.363 7.447 10.811 7 11.363 7Z"
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

export default PendingOutline;
