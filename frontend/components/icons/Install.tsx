import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInstallProps {
  color?: Colors;
  size?: IconSizes;
}

const Install = ({
  color = "ui-fleet-black-50",
  size = "medium",
}: IInstallProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g
        clipPath="url(#a)"
        fillRule="evenodd"
        clipRule="evenodd"
        fill={COLORS[color]}
      >
        <path d="M8 14A6 6 0 1 0 8 2a6 6 0 0 0 0 12Zm0 2A8 8 0 1 0 8 0a8 8 0 0 0 0 16Z" />
        <path d="M8 3.5a1 1 0 0 1 1 1v4.865l1.36-1.133a1 1 0 1 1 1.28 1.536l-3 2.5a1 1 0 0 1-1.28 0l-3-2.5a1 1 0 1 1 1.28-1.536L7 9.365V4.5a1 1 0 0 1 1-1Z" />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Install;
