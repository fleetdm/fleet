import React from "react";
import { COLORS, Colors } from "styles/var/colors";
<<<<<<< HEAD:frontend/components/icons/ChevronDown.tsx
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChevronProps {
  color?: Colors;
  size?: IconSizes;
=======
import { IconSizes, ICON_SIZES } from "styles/var/icon_sizes";

interface IChevronProps {
  color?: Colors;
  /** Default direction "down" */
  direction?: "up" | "down" | "left" | "right";
  size: IconSizes;
>>>>>>> main:frontend/components/icons/Chevron.tsx
}

const ChevronDown = ({
  color = "core-fleet-black",
<<<<<<< HEAD:frontend/components/icons/ChevronDown.tsx
=======
  direction = "down",
>>>>>>> main:frontend/components/icons/Chevron.tsx
  size = "medium",
}: IChevronProps) => {
  return (
    <svg
<<<<<<< HEAD:frontend/components/icons/ChevronDown.tsx
=======
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
>>>>>>> main:frontend/components/icons/Chevron.tsx
      xmlns="http://www.w3.org/2000/svg"
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      viewBox="0 0 16 16"
    >
      <path
        stroke={COLORS[color]}
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
        d="M12 6l-4 4-4-4"
      />
    </svg>
  );
};

export default ChevronDown;
