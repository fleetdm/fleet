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
      fill="none"
      viewBox="0 0 16 16"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M1 1a1 1 0 0 1 1-1h5.586a1 1 0 0 1 .707.293l7 7a1 1 0 0 1 0 1.414l-5.586 5.586a1 1 0 0 1-1.414 0l-7-7A1 1 0 0 1 1 6.586V1Zm4 4.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Tag;
