import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IColumnsProps {
  color?: Colors;
  size?: IconSizes;
}

const Columns = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IColumnsProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M13 1C14.381 1 15.5 2.119 15.5 3.5V12.5C15.5 13.881 14.381 15 13 15H3C1.619 15 0.5 13.881 0.5 12.5V3.5C0.5 2.119 1.619 1 3 1H13ZM11 13.5H13C13.552 13.5 14 13.052 14 12.5V3.5C14 2.948 13.552 2.5 13 2.5H11V13.5ZM6.5 13.5H9.5V2.5H6.5V13.5ZM3 2.5C2.448 2.5 2 2.948 2 3.5V12.5C2 13.052 2.448 13.5 3 13.5H5V2.5H3Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Columns;
