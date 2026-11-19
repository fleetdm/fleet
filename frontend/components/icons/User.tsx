import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IUserProps {
  color?: Colors;
  size?: IconSizes;
}

const User = ({ color = "ui-fleet-black-75", size = "medium" }: IUserProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 8.99C11.675 8.99 14.84 11.377 14.84 15.251C14.839 15.665 14.504 16.001 14.09 16.001C13.676 16.001 13.34 15.665 13.34 15.251C13.34 12.397 11.052 10.49 8 10.49C4.948 10.49 2.66 12.397 2.66 15.251C2.66 15.665 2.324 16.001 1.91 16.001C1.496 16.001 1.161 15.665 1.16 15.251C1.16 11.377 4.325 8.99 8 8.99Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 0C10.056 0 11.723 1.667 11.723 3.723C11.723 5.779 10.056 7.445 8 7.445C5.944 7.445 4.277 5.779 4.277 3.723C4.277 1.667 5.944 0 8 0ZM8 1.5C6.773 1.5 5.777 2.495 5.777 3.723C5.777 4.95 6.772 5.945 8 5.945C9.228 5.945 10.223 4.95 10.223 3.723C10.223 2.495 9.228 1.5 8 1.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default User;
