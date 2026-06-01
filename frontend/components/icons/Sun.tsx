import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ISunProps {
  color?: Colors;
  size?: IconSizes;
}

const Sun = ({ size = "medium", color = "ui-fleet-black-75" }: ISunProps) => {
  const fill = COLORS[color];
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
    >
      <circle cx="8" cy="8" r="3" fill={fill} />
      <g fill={fill}>
        <rect x="7.25" y="0.5" width="1.5" height="2.5" rx="0.75" />
        <rect x="7.25" y="13" width="1.5" height="2.5" rx="0.75" />
        <rect x="0.5" y="7.25" width="2.5" height="1.5" rx="0.75" />
        <rect x="13" y="7.25" width="2.5" height="1.5" rx="0.75" />
        <rect
          x="7.25"
          y="0.5"
          width="1.5"
          height="2.5"
          rx="0.75"
          transform="rotate(45 8 8)"
        />
        <rect
          x="7.25"
          y="0.5"
          width="1.5"
          height="2.5"
          rx="0.75"
          transform="rotate(135 8 8)"
        />
        <rect
          x="7.25"
          y="0.5"
          width="1.5"
          height="2.5"
          rx="0.75"
          transform="rotate(-45 8 8)"
        />
        <rect
          x="7.25"
          y="0.5"
          width="1.5"
          height="2.5"
          rx="0.75"
          transform="rotate(-135 8 8)"
        />
      </g>
    </svg>
  );
};

export default Sun;
