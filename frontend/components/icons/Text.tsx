import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ITextProps {
  color?: Colors;
  size?: IconSizes;
}

const Text = ({ color = "ui-fleet-black-75", size = "medium" }: ITextProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M9.25 11C9.664 11 10 11.336 10 11.75C10 12.164 9.664 12.5 9.25 12.5H3.25C2.836 12.5 2.5 12.164 2.5 11.75C2.5 11.336 2.836 11 3.25 11H9.25Z"
        fill={COLORS[color]}
      />
      <path
        d="M12.75 7.25C13.164 7.25 13.5 7.586 13.5 8C13.5 8.414 13.164 8.75 12.75 8.75H3.25C2.836 8.75 2.5 8.414 2.5 8C2.5 7.586 2.836 7.25 3.25 7.25H12.75Z"
        fill={COLORS[color]}
      />
      <path
        d="M12.75 3.5C13.164 3.5 13.5 3.836 13.5 4.25C13.5 4.664 13.164 5 12.75 5H3.25C2.836 5 2.5 4.664 2.5 4.25C2.5 3.836 2.836 3.5 3.25 3.5H12.75Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Text;
