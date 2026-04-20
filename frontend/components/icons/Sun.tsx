import React from "react";

import { Colors, COLORS } from "styles/var/colors";
import { IconSizes, ICON_SIZES } from "styles/var/icon_sizes";

interface ISun {
  color?: Colors;
  size?: IconSizes;
}

const Sun = ({ color = "ui-fleet-black-75", size = "medium" }: ISun) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <circle
        cx="8"
        cy="7.5"
        r="3"
        fill={COLORS[color]}
        stroke={COLORS[color]}
      />
      <path d="M8 0.5V2.5" stroke={COLORS[color]} strokeLinecap="round" />
      <path
        d="M15.5039 7.5L13.5039 7.5"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <path
        d="M8.00891 15.005L8.00891 13.005"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <path
        d="M13.3107 12.8048L11.8965 11.3906"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <path
        d="M2.70905 12.8058L4.12326 11.3915"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <path
        d="M2.50391 7.5L0.503906 7.5"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <path
        d="M4.11835 3.61237L2.70413 2.19815"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <path
        d="M11.9031 3.61163L13.3174 2.19742"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
    </svg>
  );
};

export default Sun;
