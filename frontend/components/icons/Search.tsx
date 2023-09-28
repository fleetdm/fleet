import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ISearchProps {
  color?: Colors;
  size?: IconSizes;
}

const Search = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: ISearchProps) => {
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
        d="M11 6.5a4.5 4.5 0 1 1-9 0 4.5 4.5 0 0 1 9 0Zm-.665 5.249a6.5 6.5 0 1 1 1.414-1.414l3.958 3.958a1 1 0 0 1-1.414 1.414l-3.958-3.958Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Search;
