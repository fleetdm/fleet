import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPendingProps {
  size?: IconSizes;
  color?: Colors;
}

const Pending = ({
  size = "medium",
  color = "ui-fleet-black-50",
}: IPendingProps) => {
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
        fillRule="evenodd"
        d="M8 16A8 8 0 108 0a8 8 0 000 16zM5 9a1 1 0 100-2 1 1 0 000 2zm4-1a1 1 0 11-2 0 1 1 0 012 0zm2 1a1 1 0 100-2 1 1 0 000 2z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Pending;
