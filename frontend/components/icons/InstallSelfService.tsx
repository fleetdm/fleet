import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInstallSelfServiceProps {
  color?: Colors;
  size?: IconSizes;
}

const InstallSelfService = ({
  color = "ui-fleet-black-50",
  size = "medium",
}: IInstallSelfServiceProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 4C8.414 4 8.75 4.335 8.75 4.75V9.937L10.659 8.029C10.952 7.736 11.426 7.736 11.719 8.029C12.012 8.322 12.012 8.796 11.719 9.089L8.53 12.28C8.481 12.328 8.426 12.366 8.369 12.398C8.341 12.413 8.315 12.43 8.286 12.442C8.254 12.455 8.22 12.464 8.187 12.472C8.174 12.476 8.161 12.481 8.148 12.484C8.142 12.485 8.137 12.486 8.131 12.487C8.088 12.494 8.045 12.5 8 12.5C7.954 12.5 7.91 12.495 7.867 12.487C7.861 12.486 7.856 12.485 7.85 12.484C7.765 12.467 7.687 12.433 7.616 12.39C7.564 12.359 7.514 12.324 7.469 12.28L4.279 9.089C3.986 8.796 3.986 8.322 4.279 8.029C4.572 7.736 5.046 7.736 5.339 8.029L7.25 9.939V4.75C7.25 4.335 7.585 4 8 4Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 0C12.418 0 16 3.581 16 8C16 12.418 12.418 16 8 16C3.581 16 0 12.418 0 8C0 3.581 3.581 0 8 0ZM8 1.5C4.41 1.5 1.5 4.41 1.5 8C1.5 11.589 4.41 14.5 8 14.5C11.589 14.5 14.5 11.589 14.5 8C14.5 4.41 11.589 1.5 8 1.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default InstallSelfService;
