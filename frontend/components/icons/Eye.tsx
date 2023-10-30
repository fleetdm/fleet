import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IEye {
  color?: Colors;
  size?: IconSizes;
}
const Eye = ({ color = "core-fleet-blue", size = "medium" }: IEye) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <g clipPath="url(#a)" fill={COLORS[color]}>
        <path d="M7.996 14C3.654 14 .246 8.6.102 8.37A.708.708 0 0 1 0 8c0-.133.036-.262.102-.37C.246 7.4 3.654 2 7.996 2c4.342 0 7.758 5.4 7.902 5.63A.708.708 0 0 1 16 8a.708.708 0 0 1-.102.37C15.754 8.6 12.346 14 7.996 14ZM2.198 8c.85 1.17 2.594 4 5.798 4 3.203 0 4.948-2.83 5.797-4-.85-1.17-2.602-4-5.797-4S3.022 6.83 2.198 8Z" />
        <path d="M6.606 10.075a2.5 2.5 0 0 0 1.387.425A2.507 2.507 0 0 0 10.5 8a2.5 2.5 0 1 0-3.894 2.075Z" />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Eye;
