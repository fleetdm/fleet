import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IRefresh {
  color?: Colors;
  size?: IconSizes;
}
const Refresh = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IRefresh) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M13.996 1.004c0-.55.45-1 1-1s.999.45.999 1v3.998c0 .55-.45 1-1 1h-3.997c-.55 0-1-.45-1-1s.45-1 1-1h1.46A5.995 5.995 0 0 0 8 2.004 6.001 6.001 0 0 0 2.004 8c0 .55-.45 1-1 1s-1-.45-1-1A7.993 7.993 0 0 1 8 .005a7.945 7.945 0 0 1 5.996 2.738V1.004Zm0 6.996c0-.55.45-1 1-1s.999.45.999 1A7.993 7.993 0 0 1 8 15.995a7.945 7.945 0 0 1-5.996-2.738v1.739c0 .55-.45.999-1 .999s-1-.45-1-1v-3.997c0-.55.45-1 1-1h3.998c.55 0 1 .45 1 1s-.45 1-1 1h-1.46A5.995 5.995 0 0 0 8 13.995 6.001 6.001 0 0 0 13.996 8Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Refresh;
