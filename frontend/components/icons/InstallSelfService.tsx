import React from "react";
import { COLORS } from "styles/var/colors";

import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInstallSelfServiceProps {
  size?: IconSizes;
  color?: keyof typeof COLORS;
}

const InstallSelfService = ({
  size = "medium",
  color = "ui-fleet-black-50",
}: IInstallSelfServiceProps) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
    >
      <g clipPath="url(#clip0_386_3648)">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8 16C12.4183 16 16 12.4183 16 8C16 3.58172 12.4183 0 8 0C3.58172 0 0 3.58172 0 8C0 12.4183 3.58172 16 8 16ZM9.00002 4C9.00002 3.44772 8.55231 3 8.00002 3C7.44774 3 7.00003 3.44772 7.00003 4V9.86496L5.64021 8.73178C5.21593 8.37821 4.58537 8.43554 4.2318 8.85982C3.87824 9.28409 3.93556 9.91466 4.35984 10.2682L7.35984 12.7682C7.73069 13.0773 8.26936 13.0773 8.64021 12.7682L11.6402 10.2682C12.0645 9.91466 12.1218 9.28409 11.7682 8.85982C11.4147 8.43554 10.7841 8.37821 10.3598 8.73178L9.00002 9.86496V4Z"
          fill={COLORS[color]}
        />
      </g>
      <defs>
        <clipPath id="clip0_386_3648">
          <rect width="16" height="16" fill="white" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default InstallSelfService;
