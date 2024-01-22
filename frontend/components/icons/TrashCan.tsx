import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { IconSizes, ICON_SIZES } from "styles/var/icon_sizes";

interface ITrashCanProps {
  color?: Colors;
  size?: IconSizes;
}

const TrashCan = ({
  color = "core-fleet-blue",
  size = "medium",
}: ITrashCanProps) => {
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
        d="M5 1a1 1 0 011-1h4a1 1 0 011 1h3a1 1 0 110 2H2a1 1 0 010-2h3zm9 3H2l.46 11.042a1 1 0 001 .958h9.08a1 1 0 001-.958L14 4zM5 6a.5.5 0 01.5.5v7a.5.5 0 01-1 0v-7A.5.5 0 015 6zm3 0a.5.5 0 01.5.5v7a.5.5 0 01-1 0v-7A.5.5 0 018 6zm3.5.5a.5.5 0 00-1 0v7a.5.5 0 001 0v-7z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default TrashCan;
