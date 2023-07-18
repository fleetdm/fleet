import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IClockProps {
  color?: Colors;
  size?: IconSizes;
}

const Clock = ({
  color = "ui-fleet-black-75",
  size = "small",
}: IClockProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 12 13"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M6 11a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9Zm0 1.5a6 6 0 1 0 0-12 6 6 0 0 0 0 12Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M6 3.125a.75.75 0 0 1 .75.75V5.75h1.125a.75.75 0 0 1 0 1.5H6a.75.75 0 0 1-.75-.75V3.875a.75.75 0 0 1 .75-.75Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Clock;
