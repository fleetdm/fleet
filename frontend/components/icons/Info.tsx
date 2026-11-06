import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInfoProps {
  color?: Colors;
  size?: IconSizes;
}

const Info = ({ color = "ui-fleet-black-75", size = "medium" }: IInfoProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 6.5C8.414 6.5 8.75 6.835 8.75 7.25V11.75C8.75 12.164 8.414 12.5 8 12.5C7.585 12.5 7.25 12.164 7.25 11.75V7.25C7.25 6.835 7.585 6.5 8 6.5Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 3.52C8.552 3.52 9 3.968 9 4.52C9 5.072 8.552 5.52 8 5.52C7.447 5.52 7 5.072 7 4.52C7 3.968 7.447 3.52 8 3.52Z"
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

export default Info;
