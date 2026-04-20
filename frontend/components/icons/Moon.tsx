import React from "react";

import { Colors, COLORS } from "styles/var/colors";
import { IconSizes, ICON_SIZES } from "styles/var/icon_sizes";

interface IMoon {
  color?: Colors;
  size?: IconSizes;
}

const Moon = ({ color = "ui-fleet-black-75", size = "medium" }: IMoon) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 14 14"
    >
      <path
        d="M4.47168 0C4.16772 0.784316 4 1.63666 4 2.52832C4 6.39431 7.13401 9.52832 11 9.52832C11.8915 9.52832 12.7432 9.3595 13.5273 9.05566C12.5135 11.6722 9.97453 13.5283 7 13.5283C3.13401 13.5283 0 10.3943 0 6.52832C0 3.55399 1.85544 1.01393 4.47168 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Moon;
