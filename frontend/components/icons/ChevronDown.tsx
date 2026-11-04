import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChevronProps {
  color?: Colors;
  size?: IconSizes;
}

const ChevronDown = ({
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
        d="M12.11 5.149C12.403 4.857 12.878 4.857 13.171 5.149C13.464 5.442 13.464 5.917 13.171 6.21L8.53 10.85C8.39 10.99 8.199 11.069 8 11.069C7.801 11.069 7.61 10.99 7.47 10.85L2.83 6.21C2.537 5.917 2.537 5.442 2.83 5.149C3.123 4.856 3.598 4.856 3.891 5.149L8 9.259L12.11 5.149Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ChevronDown;
