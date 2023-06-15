import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IColumnsProps {
  color?: Colors;
  size?: IconSizes;
}

const Columns = ({
  color = "core-fleet-blue",
  size = "medium",
}: IColumnsProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M3 2a1 1 0 0 0 0 2h3a1 1 0 0 0 0-2H3Zm0 5a1 1 0 0 0 0 2h3a1 1 0 0 0 0-2H3Zm-1 6a1 1 0 0 1 1-1h3a1 1 0 1 1 0 2H3a1 1 0 0 1-1-1Zm8-11a1 1 0 0 0 0 2h3a1 1 0 1 0 0-2h-3ZM9 8a1 1 0 0 1 1-1h3a1 1 0 1 1 0 2h-3a1 1 0 0 1-1-1Zm1 4a1 1 0 1 0 0 2h3a1 1 0 1 0 0-2h-3Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Columns;
