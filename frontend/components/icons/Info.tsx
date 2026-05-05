import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInfoProps {
  color?: Colors;
  size?: IconSizes;
}

const Info = ({ size = "small", color = "ui-fleet-black-75" }: IInfoProps) => {
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
        d="M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8Zm0 12.75a.75.75 0 0 0 .75-.75V7a.75.75 0 0 0-1.5 0v5c0 .414.336.75.75.75ZM8 3a1 1 0 1 1 0 2 1 1 0 0 1 0-2Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Info;
