import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IWarningProps {
  color?: Colors;
  size?: IconSizes;
}

const Warning = ({
  color = "status-warning",
  size = "medium",
}: IWarningProps) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
    >
      <path
        fill={COLORS[color]}
        fillRule="evenodd"
        d="M15.85 14.286l.01-.012L8.86.56l-.01.011C8.68.23 8.37 0 8 0s-.67.229-.85.571L7.14.56l-7 13.714.01.012c-.09.171-.15.354-.15.571C0 15.486.45 16 1 16h14c.55 0 1-.514 1-1.143 0-.217-.06-.4-.15-.571zM8 5.25a.75.75 0 01.75.75v4a.75.75 0 01-1.5 0V6A.75.75 0 018 5.25zM8 14a1 1 0 100-2 1 1 0 000 2z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Warning;
