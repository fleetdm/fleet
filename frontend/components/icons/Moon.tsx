import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IMoonProps {
  color?: Colors;
  size?: IconSizes;
}

const Moon = ({ size = "medium", color = "ui-fleet-black-75" }: IMoonProps) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
    >
      <path
        fill={COLORS[color]}
        d="M14 9.79A6 6 0 016.21 2a6 6 0 107.79 7.79z"
      />
    </svg>
  );
};

export default Moon;
