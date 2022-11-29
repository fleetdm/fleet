import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IExProps {
  color?: Colors;
  size?: IconSizes;
}

const Ex = ({ size = "small", color = "status-error" }: IExProps) => {
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
        d="M4.871 4.15 8 7.278l3.129-3.128a.51.51 0 0 1 .722.721L8.722 8l3.129 3.129a.51.51 0 0 1-.723.722L8 8.722l-3.129 3.129a.51.51 0 0 1-.721-.723L7.278 8 4.15 4.871a.51.51 0 0 1 .721-.721Z"
        fill={COLORS[color]}
        stroke={COLORS[color]}
      />
    </svg>
  );
};

export default Ex;
