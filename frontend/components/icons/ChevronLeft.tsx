import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChevronProps {
  color?: Colors;
  size?: IconSizes;
}

const ChevronLeft = ({
  color = "ui-fleet-black-75",
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
        d="M9.789 2.83C10.082 2.537 10.557 2.537 10.85 2.83C11.142 3.123 11.142 3.598 10.85 3.89L6.739 8L10.85 12.11C11.142 12.403 11.142 12.878 10.85 13.171C10.556 13.463 10.082 13.463 9.789 13.171L5.149 8.53C4.857 8.237 4.856 7.762 5.149 7.47L9.789 2.83Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ChevronLeft;
