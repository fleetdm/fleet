import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IM1Props {
  size?: IconSizes;
  color?: Colors;
}

const M1 = ({ size = "medium", color = "ui-fleet-black-75" }: IM1Props) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        d="M1.333 11.988H2.82V6.593h.044l2.156 5.395h1.05l2.151-5.395h.05v5.395h1.487V4.011h-1.93L5.563 9.744H5.53L3.262 4.01H1.333v7.977ZM13.014 11.988h1.653V4.011h-1.659l-2.062 1.432v1.482l2.035-1.382h.033v6.445Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default M1;
