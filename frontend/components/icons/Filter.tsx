import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IFilter {
  color?: Colors;
}
const Filter = ({ color = "core-fleet-white" }: IFilter) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <g clipPath="url(#a)">
        <path
          d="M15.35 5.922c0 2.008-.595 3.975-1.694 5.661-.499.764-1.45 1.502-2.59 2.175-1.026.605-2.138 1.12-3.066 1.532-.928-.412-2.04-.927-3.065-1.532-1.14-.673-2.092-1.41-2.59-2.175A10.375 10.375 0 0 1 .65 5.923V3.191L8 .688l7.35 2.506v2.729Z"
          stroke={COLORS[color]}
          strokeWidth="1.3"
        />
        <path
          d="M10.858 5.24 7.34 9.14l-2.309-1.5c-.33-.2-.659-.2-.879.1-.22.3-.22.6.11.8l3.297 2.2 4.287-4.8c.22-.3.22-.6-.11-.8-.22-.2-.66-.2-.88.1Z"
          fill={COLORS[color]}
        />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16.001H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Filter;
