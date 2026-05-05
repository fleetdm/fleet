import React from "react";

import { Colors, COLORS } from "styles/var/colors";
import { IconSizes, ICON_SIZES } from "styles/var/icon_sizes";

interface IPencil {
  color?: Colors;
  size?: IconSizes;
}

const Pencil = ({ color = "ui-fleet-black-75", size = "medium" }: IPencil) => {
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
        d="m14.513.507.98.98c.327.327.507.762.507 1.225 0 .463-.18.899-.508 1.226l-1.2 1.2-3.43-3.43 1.2-1.2c.654-.655 1.796-.656 2.45 0ZM1.077 11.492l8.804-8.805 3.43 3.431-8.804 8.804a.346.346 0 0 1-.16.091l-3.917.976a.343.343 0 0 1-.328-.09.348.348 0 0 1-.092-.33l.976-3.917a.35.35 0 0 1 .091-.16Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Pencil;
