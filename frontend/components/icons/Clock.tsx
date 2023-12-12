import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IClockProps {
  color?: Colors;
  size?: IconSizes;
}

const Clock = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IClockProps) => {
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
        d="M14 8A6 6 0 112 8a6 6 0 0112 0zm2 0A8 8 0 110 8a8 8 0 0116 0zM8.75 4a.75.75 0 00-1.5 0v4a.75.75 0 00.126.416l2 3a.75.75 0 101.248-.832L8.75 7.773V4z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Clock;
