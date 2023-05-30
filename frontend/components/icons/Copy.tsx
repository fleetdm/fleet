import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICopy {
  color?: Colors;
  size?: IconSizes;
}

const Copy = ({ color = "core-fleet-blue", size = "medium" }: ICopy) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        d="M2 4v9a1 1 0 0 0 1 1h9"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M6 10V2h8v8H6Z"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinejoin="round"
      />
    </svg>
  );
};
export default Copy;
