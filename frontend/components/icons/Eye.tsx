import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IEye {
  color?: Colors;
  size?: IconSizes;
}

const Eye = ({ color = "ui-fleet-black-75", size = "medium" }: IEye) => {
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
        d="M8 5.101C9.6 5.101 10.898 6.399 10.898 8C10.898 9.6 9.6 10.898 8 10.898C6.399 10.898 5.101 9.6 5.101 8C5.101 6.399 6.399 5.101 8 5.101ZM8 6.601C7.227 6.601 6.601 7.227 6.601 8C6.601 8.772 7.227 9.398 8 9.398C8.772 9.398 9.398 8.772 9.398 8C9.398 7.227 8.772 6.601 8 6.601Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 2.147C11.304 2.147 14.216 4.252 15.921 7.664C16.027 7.875 16.027 8.123 15.921 8.334C14.216 11.746 11.304 13.851 8 13.851C4.696 13.851 1.784 11.746 0.079 8.334C-0.026 8.123 -0.026 7.875 0.079 7.664C1.784 4.252 4.696 2.147 8 2.147ZM8 3.647C5.494 3.647 3.122 5.18 1.595 7.999C3.122 10.818 5.493 12.351 8 12.351C10.506 12.351 12.877 10.818 14.404 7.999C12.877 5.18 10.505 3.647 8 3.647Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Eye;
