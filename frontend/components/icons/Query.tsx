import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IQuery {
  color?: Colors;
  size?: IconSizes;
}
const Query = ({ color = "core-fleet-blue", size = "medium" }: IQuery) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M6.5 11a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9Zm0 2a6.47 6.47 0 0 0 3.835-1.251l3.958 3.958a1 1 0 0 0 1.414-1.414l-3.958-3.958A6.5 6.5 0 1 0 6.5 13Zm2.814-7.419A1 1 0 0 0 7.686 4.42l-1.814 2.54-.665-.666a1 1 0 0 0-1.414 1.414l1.5 1.5a1 1 0 0 0 1.52-.126l2.5-3.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Query;
