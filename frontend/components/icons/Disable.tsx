import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IDisable {
  color?: Colors;
  size?: IconSizes;
}

const Disable = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IDisable) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 0C12.418 0 16 3.581 16 8C16 12.418 12.418 16 8 16C3.581 16 0 12.418 0 8C0 3.581 3.581 0 8 0ZM2.904 3.965C2.025 5.074 1.5 6.475 1.5 8C1.5 11.589 4.41 14.5 8 14.5C9.524 14.5 10.924 13.973 12.033 13.094L2.904 3.965ZM8 1.5C6.475 1.5 5.073 2.026 3.964 2.905L13.093 12.034C13.973 10.925 14.5 9.524 14.5 8C14.5 4.41 11.589 1.5 8 1.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Disable;
