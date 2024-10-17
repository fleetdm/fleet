import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInstallProps {
  color?: Colors;
  size?: IconSizes;
}

const Install = ({
  color = "ui-fleet-black-50",
  size = "medium",
}: IInstallProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g clipPath="url(#clip0_798_2)">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8 14C11.3137 14 14 11.3137 14 8C14 4.68629 11.3137 2 8 2C4.68629 2 2 4.68629 2 8C2 11.3137 4.68629 14 8 14ZM8 16C12.4183 16 16 12.4183 16 8C16 3.58172 12.4183 0 8 0C3.58172 0 0 3.58172 0 8C0 12.4183 3.58172 16 8 16Z"
          fill={COLORS[color]}
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8.00003 3.5C8.55231 3.5 9.00003 3.94772 9.00003 4.5V9.36496L10.3598 8.23178C10.7841 7.87821 11.4147 7.93554 11.7682 8.35982C12.1218 8.78409 12.0645 9.41466 11.6402 9.76822L8.64021 12.2682C8.26936 12.5773 7.73069 12.5773 7.35984 12.2682L4.35984 9.76822C3.93556 9.41466 3.87824 8.78409 4.2318 8.35982C4.58537 7.93554 5.21593 7.87821 5.64021 8.23178L7.00003 9.36496V4.5C7.00003 3.94772 7.44774 3.5 8.00003 3.5Z"
          fill={COLORS[color]}
        />
      </g>
      <defs>
        <clipPath id="clip0_798_2">
          <rect width="16" height="16" fill="white" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Install;
