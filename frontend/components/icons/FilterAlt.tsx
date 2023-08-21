import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IFilterAltProps {
  color?: Colors;
  size?: IconSizes;
}

const FilterAlt = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IFilterAltProps) => {
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
        d="M5 4a1 1 0 1 0 0 2 1 1 0 0 0 0-2ZM1 4h1.17a3.001 3.001 0 0 1 5.66 0H15a1 1 0 1 1 0 2H7.83a3.001 3.001 0 0 1-5.66 0H1a1 1 0 0 1 0-2Zm0 6a1 1 0 1 0 0 2h7.17a3.001 3.001 0 0 0 5.664-.014c.054.01.11.014.166.014h1a1 1 0 1 0 0-2h-1c-.056 0-.112.005-.166.014A3.001 3.001 0 0 0 8.171 10H1Zm9 1a1 1 0 1 0 2 0 1 1 0 0 0-2 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default FilterAlt;
