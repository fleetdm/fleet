import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ISettingsProps {
  color?: Colors;
  size?: IconSizes;
}

const Settings = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: ISettingsProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M6.39.876A1 1 0 0 1 7.384 0h1.234a1 1 0 0 1 .992.876l.175 1.394c.35.109.687.249 1.006.417l1.11-.863a1 1 0 0 1 1.32.083l.873.873a1 1 0 0 1 .082 1.32l-.862 1.11c.168.32.308.656.417 1.006l1.394.175a1 1 0 0 1 .876.992v1.234a1 1 0 0 1-.876.992l-1.394.175c-.109.35-.249.687-.417 1.006l.862 1.11a1 1 0 0 1-.082 1.32l-.873.873a1 1 0 0 1-1.32.082l-1.11-.862c-.32.168-.656.308-1.006.417l-.175 1.394a1 1 0 0 1-.992.876H7.383a1 1 0 0 1-.992-.876l-.175-1.394a5.959 5.959 0 0 1-1.006-.417l-1.11.862a1 1 0 0 1-1.32-.082l-.873-.873a1 1 0 0 1-.083-1.32l.863-1.11a5.962 5.962 0 0 1-.417-1.006L.876 9.609A1 1 0 0 1 0 8.617V7.383a1 1 0 0 1 .876-.992l1.394-.175c.109-.35.249-.687.417-1.006L1.824 4.1a1 1 0 0 1 .083-1.32l.873-.873a1 1 0 0 1 1.32-.083l1.11.863c.32-.168.656-.308 1.006-.417L6.391.876ZM4 8a4 4 0 1 0 6.831-2.826l-.005-.005A4 4 0 0 0 4 8Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Settings;
