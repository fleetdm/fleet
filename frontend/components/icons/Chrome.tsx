import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChromeProps {
  size: IconSizes;
  color?: Colors;
}

const Chrome = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IChromeProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        d="M0 8a8.14 8.14 0 0 1 1.07-4.028l3.433 5.975A4.001 4.001 0 0 0 8 12c.447 0 .847-.072 1.275-.206l-2.384 4.131A8.004 8.004 0 0 1 0 8Zm11.41 2.05c.384-.6.59-1.328.59-2.05a4.001 4.001 0 0 0-1.353-3h4.772c.375.925.581 1.94.581 3 0 4.419-3.581 7.972-8 8l3.41-5.95ZM14.93 4H8a3.98 3.98 0 0 0-3.922 3.21L1.693 3.076A7.983 7.983 0 0 1 8 0a7.999 7.999 0 0 1 6.931 4ZM5.25 8a2.75 2.75 0 1 1 5.5 0 2.75 2.75 0 0 1-5.5 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Chrome;
