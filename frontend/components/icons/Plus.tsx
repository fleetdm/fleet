import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPlusProps {
  color?: Colors;
  size?: IconSizes;
}

const Plus = ({ color = "ui-fleet-black-75", size = "medium" }: IPlusProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      aria-label="plus"
    >
      <path
        d="M8 3v10M3 8h10"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
};

export default Plus;
