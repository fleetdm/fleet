import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IEye {
  color?: Colors;
  size?: IconSizes;
}
const Eye = ({ color = "ui-fleet-black-75", size = "medium" }: IEye) => {
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
        d="M8 2C3.384 2 .895 5.82.171 7.629a1 1 0 000 .742C.895 10.18 3.384 14 8 14s7.105-3.82 7.829-5.629a1 1 0 000-.742C15.104 5.82 12.616 2 8 2zm0 10c-3.125 0-5.05-2.427-5.808-4C2.595 7.163 3.33 6.083 4.4 5.256A4 4 0 007.99 11 4.011 4.011 0 0012 7a4 4 0 00-.4-1.743c1.07.826 1.805 1.906 2.207 2.743-.756 1.573-2.682 4-5.807 4zm.664-4.955a1.5 1.5 0 101.672-2.49 1.5 1.5 0 00-1.672 2.49z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Eye;
