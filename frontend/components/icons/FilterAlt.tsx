import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IFilterAlt {
  color?: Colors;
}
const FilterAlt = ({ color = "core-fleet-white" }: IFilterAlt) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M4 3a1 1 0 1 1 2 0 1 1 0 0 1-2 0ZM2.17 2H1a1 1 0 0 0 0 2h1.17a3.001 3.001 0 0 0 5.66 0H15a1 1 0 1 0 0-2H7.83a3.001 3.001 0 0 0-5.66 0ZM0 13a1 1 0 0 1 1-1h2.17a3.001 3.001 0 0 1 5.664.014c.054-.01.11-.014.166-.014h6a1 1 0 1 1 0 2H9c-.057 0-.112-.005-.166-.014A3.001 3.001 0 0 1 3.171 14H1a1 1 0 0 1-1-1Zm6 1a1 1 0 1 1 0-2 1 1 0 0 1 0 2ZM1 7a1 1 0 0 0 0 2h7c.057 0 .112-.005.166-.014A3.001 3.001 0 0 0 13.829 9H15a1 1 0 1 0 0-2h-1.17a3.001 3.001 0 0 0-5.664.014A1.007 1.007 0 0 0 8 7H1Zm11 1a1 1 0 1 0-2 0 1 1 0 0 0 2 0Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default FilterAlt;
