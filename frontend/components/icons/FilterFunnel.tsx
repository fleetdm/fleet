import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IFilterFunnelProps {
  color?: Colors;
  size?: IconSizes;
}

const FilterFunnel = ({
  size = "medium",
  color = "ui-fleet-black-33",
}: IFilterFunnelProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m15.86 1.497-.01-.01A.934.934 0 0 0 16 .991.999.999 0 0 0 15 0H1C.45 0 0 .446 0 .991c0 .189.06.347.15.496l-.01.01L6 7.371V14l4 2V7.371l5.86-5.874Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default FilterFunnel;
