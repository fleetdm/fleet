import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IQuery {
  color?: Colors;
  size?: IconSizes;
}
const Query = ({ color = "ui-fleet-black-75", size = "medium" }: IQuery) => {
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
        d="M6.5 11a4.5 4.5 0 100-9 4.5 4.5 0 000 9zm0 2a6.47 6.47 0 003.835-1.251l3.958 3.958a1 1 0 001.414-1.414l-3.958-3.958A6.5 6.5 0 106.5 13zm2.61-7.564a.75.75 0 10-1.22-.872l-1.986 2.78-.874-.874a.75.75 0 00-1.06 1.06l1.5 1.5a.75.75 0 001.14-.094l2.5-3.5z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Query;
