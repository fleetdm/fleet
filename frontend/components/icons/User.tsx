import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IUserProps {
  size?: IconSizes;
  color?: Colors;
}

const User = ({ size = "medium", color = "ui-fleet-black-75" }: IUserProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M3.13 12a8.87 8.87 0 1 1 16.043 5.218 7.824 7.824 0 0 0-2.926-3.444 5.74 5.74 0 1 0-8.493 0 7.82 7.82 0 0 0-2.927 3.444A8.83 8.83 0 0 1 3.13 12Zm4.24 7.567A8.828 8.828 0 0 0 12 20.87a8.829 8.829 0 0 0 4.63-1.303 4.696 4.696 0 0 0-9.26 0ZM12 0C5.373 0 0 5.373 0 12s5.373 12 12 12 12-5.373 12-12S18.627 0 12 0ZM9.391 9.914a2.609 2.609 0 1 1 5.218 0 2.609 2.609 0 0 1-5.218 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default User;
