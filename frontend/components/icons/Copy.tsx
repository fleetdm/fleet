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
        d="M2.75 0.5H8.75Q11 0.5 11 2.75V8.75Q11 11 8.75 11H2.75Q1.818 11 1.159 10.341Q0.5 9.682 0.5 8.75V2.75Q0.5 1.818 1.159 1.159Q1.818 0.5 2.75 0.5ZM2.75 2Q2 2 2 2.75V8.75Q2 9.5 2.75 9.5H8.75Q9.5 9.5 9.5 8.75V2.75Q9.5 2 8.75 2Z"
        fill={COLORS[color]}
      />
      <path
        d="M6.75 13.25Q6.75 14 7.5 14H13.5Q14.25 14 14.25 13.25V7.25Q14.25 6.5 13.5 6.5H13.25Q12.939 6.5 12.72 6.28Q12.5 6.061 12.5 5.75Q12.5 5.439 12.72 5.22Q12.939 5 13.25 5H13.5Q15.75 5 15.75 7.25V13.25Q15.75 15.5 13.5 15.5H7.5Q5.25 15.5 5.25 13.25Q5.25 12.939 5.47 12.72Q5.689 12.5 6 12.5Q6.311 12.5 6.53 12.72Q6.75 12.939 6.75 13.25Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Copy;
