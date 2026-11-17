import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IArrowLeftProps {
  color?: Colors;
  size?: IconSizes;
}

const ArrowLeft = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IArrowLeftProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M6.565 3.625C6.858 3.332 7.333 3.332 7.626 3.625C7.919 3.918 7.919 4.393 7.626 4.685L5.061 7.25H12.75C13.164 7.25 13.5 7.586 13.5 8C13.5 8.414 13.164 8.75 12.75 8.75H5.061L7.626 11.315C7.919 11.608 7.919 12.083 7.626 12.376C7.333 12.668 6.858 12.668 6.565 12.376L2.72 8.531C2.665 8.476 2.622 8.412 2.587 8.345C2.533 8.241 2.5 8.125 2.5 8C2.5 7.793 2.584 7.605 2.72 7.47L6.565 3.625Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ArrowLeft;
