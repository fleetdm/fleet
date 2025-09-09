import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ILightbulbProps {
  color?: Colors;
  size?: IconSizes;
}

const Lightbulb = ({
  size = "medium",
  color = "ui-fleet-black-75",
}: ILightbulbProps) => {
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
        d="M3 4.35C3 1.95 5.23 0 7.98 0c2.76 0 4.99 1.95 4.99 4.35 0 1.22-.432 1.884-.894 2.593-.435.667-.896 1.375-1.046 2.627 0 .25-.22.44-.5.44H5.44c-.28 0-.5-.2-.5-.44-.15-1.252-.611-1.96-1.046-2.627C3.432 6.233 3 5.57 3 4.35zM10.98 12c0-.55-.44-1-1-1h-4c-.55 0-1 .45-1 1s.45 1 1 1h4c.55 0 1-.45 1-1zm-1 3c0-.55-.44-1-1-1h-2c-.55 0-1 .45-1 1s.45 1 1 1h2c.55 0 1-.45 1-1z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Lightbulb;
