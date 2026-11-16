import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IWindowsProps {
  color?: Colors;
  size?: IconSizes;
}

const Windows = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IWindowsProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M14.424 8.283V15.138L6.943 13.657V8.283H14.424Z"
        fill={COLORS[color]}
      />
      <path
        d="M6.206 8.283V13.513L1.572 12.587V8.283H6.206Z"
        fill={COLORS[color]}
      />
      <path
        d="M6.206 7.699H1.572V3.422L6.206 2.496V7.699Z"
        fill={COLORS[color]}
      />
      <path
        d="M14.425 7.699H6.943V2.354L14.425 0.857V7.699Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Windows;
