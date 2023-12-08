import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IExProps {
  color?: Colors;
  size?: IconSizes;
}

const Close = ({ size = "medium", color = "core-fleet-blue" }: IExProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M3 3L13 13M3 13L13 3"
        stroke={COLORS[color]}
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
};

export default Close;
