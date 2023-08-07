import React from "react";
import { Colors, COLORS } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IWindowsCircledProps {
  size: IconSizes;
  iconColor?: Colors;
  bgColor?: Colors;
}

const WindowsCircled = ({
  size = "extra-large",
  iconColor = "ui-fleet-black-75", // default grey
  bgColor = "ui-blue-10", // default light blue
}: IWindowsCircledProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="24" cy="24" r="24" fill={COLORS[bgColor]} />
      <rect
        width="24"
        height="24"
        transform="translate(12 12)"
        fill={COLORS[bgColor]}
      />
      <path
        d="M13.0918 31.7125L20.8792 33.2694V24.479H13.0918V31.7125Z"
        fill={COLORS[iconColor]}
      />
      <path
        d="M13.0918 23.4969H20.8792V14.7544L13.0918 16.3113V23.4969Z"
        fill={COLORS[iconColor]}
      />
      <path
        d="M22.1172 23.497H34.6914V12L22.1172 14.515V23.497Z"
        fill={COLORS[iconColor]}
      />
      <path
        d="M22.1172 33.5089L34.6914 36V24.479H22.1172V33.5089Z"
        fill={COLORS[iconColor]}
      />
    </svg>
  );
};

export default WindowsCircled;
