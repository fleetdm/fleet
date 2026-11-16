import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICopy {
  color?: Colors;
  size?: IconSizes;
}

const Copy = ({ color = "ui-fleet-black-75", size = "medium" }: ICopy) => {
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
        d="M2.75 0.5H8.75Q11 0.5 11 2.75V9.25Q11 11.5 8.75 11.5H2.75Q1.818 11.5 1.159 10.841Q0.5 10.182 0.5 9.25V2.75Q0.5 1.818 1.159 1.159Q1.818 0.5 2.75 0.5ZM2.75 2Q2 2 2 2.75V9.25Q2 10 2.75 10H8.75Q9.5 10 9.5 9.25V2.75Q9.5 2 8.75 2Z"
        fill={COLORS[color]}
      />
      <path
        d="M6.793 13.5Q6.97 14 7.5 14H13.5Q14.25 14 14.25 13.25V6.75Q14.25 6 13.5 6H13.25Q12.939 6 12.72 5.78Q12.5 5.561 12.5 5.25Q12.5 4.939 12.72 4.72Q12.939 4.5 13.25 4.5H13.5Q15.75 4.5 15.75 6.75V13.25Q15.75 15.5 13.5 15.5H7.5Q5.909 15.5 5.379 14Q5.275 13.707 5.409 13.427Q5.543 13.146 5.836 13.043Q6.129 12.939 6.409 13.073Q6.69 13.207 6.793 13.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Copy;
