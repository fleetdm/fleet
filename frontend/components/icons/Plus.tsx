import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPlusProps {
  color?: Colors;
  size?: IconSizes;
}

const Plus = ({ color = "ui-fleet-black-75", size = "medium" }: IPlusProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-label="plus"
    >
      <path
        d="M7.25 12.75V8.75H3.25C2.836 8.75 2.5 8.414 2.5 8C2.5 7.586 2.836 7.25 3.25 7.25H7.25V3.25C7.25 2.836 7.586 2.5 8 2.5C8.414 2.5 8.75 2.836 8.75 3.25V7.25H12.75C13.164 7.25 13.5 7.586 13.5 8C13.5 8.414 13.164 8.75 12.75 8.75H8.75V12.75C8.75 13.164 8.414 13.5 8 13.5C7.586 13.5 7.25 13.164 7.25 12.75Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Plus;
