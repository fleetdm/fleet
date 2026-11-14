import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IMoreProps {
  color?: Colors;
  size?: IconSizes;
}

const More = ({ color = "ui-fleet-black-75", size = "medium" }: IMoreProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 12.179C8.69 12.179 9.25 12.738 9.25 13.429C9.25 14.119 8.69 14.679 8 14.679C7.31 14.679 6.75 14.119 6.75 13.429C6.75 12.738 7.31 12.179 8 12.179Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 6.75C8.69 6.75 9.25 7.31 9.25 8C9.25 8.69 8.69 9.25 8 9.25C7.31 9.25 6.75 8.69 6.75 8C6.75 7.31 7.31 6.75 8 6.75Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 1.321C8.69 1.321 9.25 1.881 9.25 2.571C9.25 3.262 8.69 3.821 8 3.821C7.31 3.821 6.75 3.262 6.75 2.571C6.75 1.881 7.31 1.321 8 1.321Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default More;
