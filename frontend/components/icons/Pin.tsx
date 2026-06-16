import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPinProps {
  color?: Colors;
  size?: IconSizes;
}

const Pin = ({ color = "ui-fleet-black-75", size = "medium" }: IPinProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.293.293a1 1 0 0 1 1.414 0l5 5a1 1 0 0 1-1.414 1.414l-.293-.293-2.793 2.793a3 3 0 0 1-.879 2.328l-.586.586a1 1 0 0 1-1.414 0L6.5 9.914l-4.793 4.793a1 1 0 0 1-1.414-1.414L5.086 8.5 3.172 6.586a1 1 0 0 1 0-1.414l.586-.586a3 3 0 0 1 2.328-.879L8.879 1A1 1 0 0 1 9.293.293Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Pin;
