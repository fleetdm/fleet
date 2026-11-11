import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IChromeProps {
  color?: Colors;
  size?: IconSizes;
}

const Chrome = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IChromeProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M15.419 5C15.794 5.925 16 6.94 16 8C16 12.419 12.419 15.972 8 16L11.411 10.05C11.795 9.45 12.001 8.722 12.001 8C12.001 7.433 11.88 6.872 11.647 6.354C11.414 5.837 11.073 5.375 10.648 5H15.419Z"
        fill={COLORS[color]}
      />
      <path
        d="M4.503 9.947C4.85 10.57 5.358 11.088 5.972 11.449C6.587 11.81 7.288 12 8 12C8.447 12 8.848 11.928 9.276 11.794L6.891 15.925C4.98 15.656 3.231 14.707 1.964 13.251C0.698 11.795 0.001 9.93 0 8C0.002 6.587 0.371 5.199 1.071 3.972L4.503 9.947Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 5.25C8.73 5.25 9.429 5.54 9.945 6.056C10.46 6.571 10.751 7.271 10.751 8C10.751 8.729 10.46 9.429 9.945 9.944C9.429 10.46 8.73 10.75 8 10.75C7.271 10.75 6.572 10.46 6.056 9.944C5.54 9.429 5.25 8.729 5.25 8C5.25 7.271 5.54 6.571 6.056 6.056C6.572 5.54 7.271 5.25 8 5.25Z"
        fill={COLORS[color]}
      />
      <path
        d="M8 0C9.405 -0.001 10.786 0.369 12.002 1.071C13.219 1.773 14.228 2.784 14.93 4H8C7.076 3.996 6.178 4.315 5.462 4.9C4.747 5.486 4.258 6.303 4.079 7.21L1.694 3.076C2.441 2.117 3.398 1.341 4.491 0.808C5.584 0.275 6.784 -0.002 8 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Chrome;
