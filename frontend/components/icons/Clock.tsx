import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IClockProps {
  color?: Colors;
  size?: IconSizes;
}

const Clock = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IClockProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 3.5C8.414 3.5 8.75 3.835 8.75 4.25V7.618L11.026 9.135C11.37 9.365 11.463 9.831 11.234 10.175C11.004 10.52 10.539 10.613 10.194 10.383L7.583 8.643C7.375 8.504 7.25 8.27 7.25 8.019V4.25C7.25 3.835 7.585 3.5 8 3.5Z"
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

export default Clock;
