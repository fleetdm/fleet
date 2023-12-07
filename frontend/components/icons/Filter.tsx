import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IFilterProps {
  color?: Colors;
  size?: IconSizes;
}

const Filter = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IFilterProps) => {
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
        d="M2 3a1 1 0 0 1 1-1h10a1 1 0 1 1 0 2H3a1 1 0 0 1-1-1Zm2 10a1 1 0 0 1 1-1h6a1 1 0 1 1 0 2H5a1 1 0 0 1-1-1Zm0-6a1 1 0 0 0 0 2h8a1 1 0 1 0 0-2H4Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Filter;
