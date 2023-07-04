import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IPendingProps {
  size?: IconSizes;
  color?: Colors;
}

const Pending = ({
  size = "medium",
  color = "ui-fleet-black-50",
}: IPendingProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 17"
      fill="none"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 16.5C12.4183 16.5 16 12.9183 16 8.5C16 4.08172 12.4183 0.5 8 0.5C3.58172 0.5 0 4.08172 0 8.5C0 12.9183 3.58172 16.5 8 16.5ZM4.6665 9.5C5.21879 9.5 5.6665 9.05229 5.6665 8.5C5.6665 7.94772 5.21879 7.5 4.6665 7.5C4.11422 7.5 3.6665 7.94772 3.6665 8.5C3.6665 9.05229 4.11422 9.5 4.6665 9.5ZM8.6665 8.5C8.6665 9.05229 8.21879 9.5 7.6665 9.5C7.11422 9.5 6.6665 9.05229 6.6665 8.5C6.6665 7.94772 7.11422 7.5 7.6665 7.5C8.21879 7.5 8.6665 7.94772 8.6665 8.5ZM10.6665 9.5C11.2188 9.5 11.6665 9.05229 11.6665 8.5C11.6665 7.94772 11.2188 7.5 10.6665 7.5C10.1142 7.5 9.6665 7.94772 9.6665 8.5C9.6665 9.05229 10.1142 9.5 10.6665 9.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Pending;
