import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ITagProps {
  color?: Colors;
  size?: IconSizes;
}

const Tag = ({ color = "ui-fleet-black-75", size = "medium" }: ITagProps) => {
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
        d="M.609 8.04a2.08 2.08 0 0 0 0 2.94l4.41 4.411a2.08 2.08 0 0 0 2.94 0l7.432-7.431c.417-.417.637-.991.606-1.58l-.27-5.124a1.04 1.04 0 0 0-.983-.983L9.62.003a2.08 2.08 0 0 0-1.58.606zM10.9 5.1a1.56 1.56 0 1 0 2.205-2.205A1.56 1.56 0 0 0 10.9 5.1"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Tag;
