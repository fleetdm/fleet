import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPinProps {
  color?: Colors;
  size?: IconSizes;
}

const Pin = ({ color = "ui-fleet-black-75", size = "medium" }: IPinProps) => {
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
        d="M12.724.346a1.18 1.18 0 0 0-1.667 0L7.722 3.68a.83.83 0 0 1-.864.194l-1.695-.618a1.1 1.1 0 0 0-1.15.254L2.834 4.689c-.46.46-.46 1.206 0 1.667l2.573 2.57-5.061 5.061a1.178 1.178 0 1 0 1.667 1.667l5.06-5.06 2.573 2.572c.46.46 1.206.46 1.667 0l1.178-1.18c.301-.299.4-.748.254-1.149l-.618-1.695a.83.83 0 0 1 .195-.864l3.332-3.335c.461-.46.461-1.206 0-1.667z"
        clipRule="evenodd"
      />
    </svg>
  );
};

export default Pin;
