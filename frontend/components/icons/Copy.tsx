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
        d="M8.75 4.5C9.992 4.5 11 5.507 11 6.75V13.25C11 14.492 9.992 15.499 8.75 15.5H2.75C1.507 15.5 0.5 14.492 0.5 13.25V6.75C0.5 5.557 1.428 4.584 2.601 4.507C2.634 4.503 2.667 4.5 2.701 4.5H8.75ZM2.75 6C2.335 6 2 6.335 2 6.75V13.25C2 13.664 2.335 14 2.75 14H8.75C9.164 13.999 9.5 13.664 9.5 13.25V6.75C9.5 6.335 9.164 6 8.75 6H2.75Z"
        fill={COLORS[color]}
      />
      <path
        d="M13.249 0.5C14.491 0.5 15.499 1.507 15.499 2.75V10.25C15.498 10.664 15.163 11 14.749 11C14.335 10.999 13.999 10.664 13.999 10.25V2.75C13.999 2.335 13.663 2 13.249 2H5.25C4.835 2 4.5 1.664 4.5 1.25C4.5 0.835 4.835 0.5 5.25 0.5H13.249Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Copy;
