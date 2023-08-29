import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IWindowsProps {
  size: IconSizes;
  color?: Colors;
}

const Windows = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IWindowsProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        d="m1.092 13.142 5.192 1.038V8.32H1.092v4.822ZM1.092 7.665h5.192V1.836L1.092 2.874v4.79ZM7.11 7.665h8.382V0L7.11 1.677v5.988ZM7.11 14.34 15.491 16V8.32H7.11v6.02Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Windows;
