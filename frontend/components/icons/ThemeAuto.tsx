import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IThemeAutoProps {
  color?: Colors;
  size?: IconSizes;
}

// Half-filled circle: outline ring + filled left semicircle. Reads as "auto"
// in a segmented control between a sun and a moon — neither fully light nor
// fully dark.
const ThemeAuto = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: IThemeAutoProps) => {
  const fill = COLORS[color];
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
    >
      <path
        fill={fill}
        fillRule="evenodd"
        d="M8 1.5a6.5 6.5 0 100 13 6.5 6.5 0 000-13zM8 13a5 5 0 110-10 5 5 0 010 10z"
        clipRule="evenodd"
      />
      <path fill={fill} d="M8 3a5 5 0 000 10V3z" />
    </svg>
  );
};

export default ThemeAuto;
