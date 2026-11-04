import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IExCircledProps {
  color?: Colors;
  size?: IconSizes;
}

const CloseFilled = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IExCircledProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M11.315 3.625C11.608 3.332 12.083 3.332 12.376 3.625C12.669 3.918 12.669 4.393 12.376 4.685L9.06 8L12.376 11.315C12.669 11.608 12.669 12.083 12.376 12.376C12.083 12.669 11.608 12.669 11.315 12.376L8 9.06L4.685 12.376C4.393 12.669 3.918 12.669 3.625 12.376C3.332 12.083 3.332 11.608 3.625 11.315L6.939 8L3.625 4.685C3.332 4.393 3.332 3.918 3.625 3.625C3.918 3.332 4.393 3.332 4.685 3.625L8 6.939L11.315 3.625Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default CloseFilled;
