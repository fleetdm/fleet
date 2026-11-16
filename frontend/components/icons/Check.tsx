import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICheckProps {
  color?: Colors;
  size?: IconSizes;
}

const Check = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: ICheckProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-label="check"
    >
      <path
        d="M12.202 4.321C12.485 4.018 12.959 4.002 13.262 4.285C13.564 4.567 13.58 5.042 13.298 5.344L6.965 12.13C6.826 12.279 6.633 12.365 6.43 12.369C6.226 12.372 6.03 12.293 5.887 12.149L2.72 8.982C2.427 8.689 2.427 8.214 2.72 7.922C3.013 7.629 3.487 7.629 3.78 7.922L6.397 10.539L12.202 4.321Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Check;
