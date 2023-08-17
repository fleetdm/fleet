import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IWarningProps {
  color?: Colors;
  size?: IconSizes;
}

const Warning = ({
  color = "status-warning",
  size = "small",
}: IWarningProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 12 13"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m11.887 11.214.008-.008L6.645.92l-.007.009C6.51.67 6.277.5 6 .5c-.277 0-.503.171-.638.429L5.356.92.105 11.206l.008.008a.898.898 0 0 0-.113.429c0 .471.338.857.75.857h10.5c.412 0 .75-.386.75-.857 0-.163-.045-.3-.113-.429ZM6 4.25a.75.75 0 0 1 .75.75v3a.75.75 0 0 1-1.5 0V5A.75.75 0 0 1 6 4.25ZM6 11a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Warning;
