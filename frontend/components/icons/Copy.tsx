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
        d="M2.702 0.5H8.75C10.243 0.5 11 1.257 11 2.75V9.25C11 10.743 10.243 11.5 8.75 11.5H2.75C1.257 11.5 0.5 10.743 0.5 9.25V2.75C0.5 1.286 1.221 0.529 2.663 0.5C2.676 0.5 2.689 0.5 2.702 0.5ZM2.75 2C2.25 2 2 2.25 2 2.75V9.25C2 9.75 2.25 10 2.75 10H8.75C9.25 10 9.5 9.75 9.5 9.25V2.75C9.5 2.25 9.25 2 8.75 2H2.75Z"
        fill={COLORS[color]}
      />
      <path
        d="M6.75 13.25C6.75 13.75 7 14 7.5 14H13.5C14 14 14.25 13.75 14.25 13.25V6.75C14.25 6.25 14 6 13.5 6H13C12.586 6 12.25 5.664 12.25 5.25C12.25 4.836 12.586 4.5 13 4.5H13.5C14.993 4.5 15.75 5.257 15.75 6.75V13.25C15.75 14.743 14.993 15.5 13.5 15.5H7.5C6.007 15.5 5.25 14.743 5.25 13.25C5.25 12.836 5.586 12.5 6 12.5C6.414 12.5 6.75 12.836 6.75 13.25Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Copy;
