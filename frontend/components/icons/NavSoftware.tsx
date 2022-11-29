import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface INavSoftware {
  color?: Colors;
}
const NavSoftware = ({ color = "core-fleet-white" }: INavSoftware) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <circle cx="8" cy="8" r="7.5" stroke={COLORS[color]} />
      <path
        d="M3 8a5 5 0 0 1 5-5"
        stroke={COLORS[color]}
        strokeLinecap="round"
      />
      <circle cx="8" cy="8" r="1.5" stroke={COLORS[color]} />
    </svg>
  );
};

export default NavSoftware;
