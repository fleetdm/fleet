import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ISearchProps {
  color?: Colors;
  size?: IconSizes;
}

const Search = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: ISearchProps) => {
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
        d="M6.826 0.5C10.32 0.5 13.152 3.332 13.152 6.826C13.152 8.302 12.645 9.659 11.797 10.735L15.281 14.22C15.574 14.513 15.574 14.987 15.281 15.28C14.988 15.573 14.514 15.573 14.221 15.28L10.736 11.796C9.66 12.644 8.303 13.152 6.826 13.152C3.332 13.152 0.5 10.32 0.5 6.826C0.5 3.332 3.332 0.5 6.826 0.5ZM6.826 2C4.161 2 2 4.161 2 6.826C2 9.492 4.161 11.652 6.826 11.652C9.492 11.652 11.652 9.492 11.652 6.826C11.652 4.161 9.492 2 6.826 2Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Search;
