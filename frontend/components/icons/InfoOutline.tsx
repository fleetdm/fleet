import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IInfoOutlineProps {
  size?: IconSizes;
  color?: Colors;
}

const InfoOutline = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IInfoOutlineProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 17 17"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8.5 14.5C11.8137 14.5 14.5 11.8137 14.5 8.5C14.5 5.18629 11.8137 2.5 8.5 2.5C5.18629 2.5 2.5 5.18629 2.5 8.5C2.5 11.8137 5.18629 14.5 8.5 14.5ZM8.5 16.5C12.9183 16.5 16.5 12.9183 16.5 8.5C16.5 4.08172 12.9183 0.5 8.5 0.5C4.08172 0.5 0.5 4.08172 0.5 8.5C0.5 12.9183 4.08172 16.5 8.5 16.5ZM8.5 12.5C7.94772 12.5 7.5 12.0523 7.5 11.5L7.5 8.5C7.5 7.94772 7.94771 7.5 8.5 7.5C9.05228 7.5 9.5 7.94772 9.5 8.5V11.5C9.5 12.0523 9.05229 12.5 8.5 12.5ZM8.5 4.5C7.94772 4.5 7.5 4.94772 7.5 5.5C7.5 6.05228 7.94772 6.5 8.5 6.5C9.05229 6.5 9.5 6.05228 9.5 5.5C9.5 4.94772 9.05228 4.5 8.5 4.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default InfoOutline;
