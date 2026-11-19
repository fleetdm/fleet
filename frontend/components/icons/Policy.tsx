import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPolicyProps {
  color?: Colors;
  size?: IconSizes;
}

const Policy = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IPolicyProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M10.062 5.458C10.345 5.155 10.82 5.14 11.122 5.423C11.424 5.706 11.44 6.181 11.157 6.483L7.793 10.079C7.654 10.227 7.461 10.313 7.258 10.316C7.055 10.319 6.858 10.24 6.715 10.097L4.859 8.241C4.567 7.948 4.567 7.473 4.859 7.18C5.152 6.888 5.627 6.888 5.92 7.18L7.228 8.487L10.062 5.458Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M7.75 0.043C7.912 -0.014 8.088 -0.014 8.25 0.043L14 2.073C14.3 2.179 14.5 2.462 14.5 2.78V7.42C14.5 11.278 11.827 14.403 8.299 15.937C8.108 16.02 7.892 16.02 7.701 15.937C4.173 14.403 1.5 11.278 1.5 7.42V2.78C1.5 2.462 1.7 2.179 2 2.073L7.75 0.043ZM3 3.31V7.42C3 10.422 5.021 13.024 8 14.427C10.979 13.024 13 10.422 13 7.42V3.31L8 1.545L3 3.31Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Policy;
