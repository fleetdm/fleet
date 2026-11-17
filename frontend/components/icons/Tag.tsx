import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ITagProps {
  color?: Colors;
  size?: IconSizes;
}

const Tag = ({ color = "ui-fleet-black-75", size = "medium" }: ITagProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M4.771 3.521C5.462 3.521 6.021 4.081 6.021 4.771C6.021 5.462 5.462 6.021 4.771 6.021C4.081 6.021 3.521 5.462 3.521 4.771C3.521 4.081 4.081 3.521 4.771 3.521Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M7.099 0.5C7.39 0.5 7.678 0.558 7.947 0.669L8.145 0.762C8.336 0.864 8.512 0.995 8.666 1.149L14.22 6.702L14.374 6.873C15.095 7.757 15.044 9.06 14.22 9.884L9.884 14.22C9.005 15.098 7.581 15.099 6.702 14.22L1.149 8.667C0.995 8.513 0.864 8.336 0.762 8.145L0.669 7.947C0.558 7.678 0.5 7.39 0.5 7.099V2.75C0.5 1.549 1.441 0.572 2.625 0.507C2.655 0.503 2.686 0.5 2.718 0.5H7.099ZM2.75 2C2.336 2 2 2.336 2 2.75V7.099C2 7.193 2.019 7.286 2.055 7.373L2.121 7.497C2.147 7.536 2.176 7.572 2.209 7.605L7.763 13.159C8.055 13.452 8.53 13.452 8.823 13.159L13.159 8.823C13.434 8.549 13.451 8.115 13.211 7.82L13.159 7.764L7.605 2.21C7.572 2.177 7.536 2.147 7.497 2.121L7.373 2.055C7.286 2.019 7.193 2 7.099 2H2.75Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Tag;
