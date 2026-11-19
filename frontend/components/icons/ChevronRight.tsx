import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChevronProps {
  color?: Colors;
  size?: IconSizes;
}

const ChevronRight = ({
  color = "core-fleet-black",
  size = "medium",
}: IChevronProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M5.149 2.83C5.442 2.537 5.917 2.537 6.21 2.83L10.85 7.47C11.142 7.763 11.142 8.237 10.85 8.53L6.21 13.171C5.917 13.464 5.442 13.464 5.149 13.171C4.857 12.878 4.857 12.403 5.149 12.11L9.259 8L5.149 3.891C4.856 3.598 4.856 3.123 5.149 2.83Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ChevronRight;
