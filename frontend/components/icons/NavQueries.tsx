import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface INavQuery {
  color?: Colors;
}
const NavQuery = ({ color = "core-fleet-white" }: INavQuery) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <g clipPath="url(#a)" fill={COLORS[color]}>
        <path d="m15.4 12.5-2.3-2.2c.6-.9.9-2.1.9-3.3 0-3.9-3.1-7-7-7S0 3.1 0 7s3.1 7 7 7c1.2 0 2.3-.3 3.2-.8l2.3 2.2c.4.4.9.6 1.4.6.5 0 1-.2 1.4-.6.9-.8.9-2.1.1-2.9ZM1.2 7c0-3.2 2.6-5.8 5.8-5.8 3.2 0 5.8 2.6 5.8 5.8 0 3.2-2.6 5.8-5.8 5.8-3.2 0-5.8-2.6-5.8-5.8Zm13.4 7.6c-.3.3-.8.3-1.2 0l-2.1-2c.4-.3.8-.7 1.2-1.2l2.1 2c.3.3.3.8 0 1.2Z" />
        <path d="M9.6 4.3 6.4 8.2 4.3 6.7c-.3-.2-.6-.2-.8.1-.2.3-.2.6.1.8l3 2.2L10.5 5c.2-.3.2-.6-.1-.8-.2-.2-.6-.2-.8.1Z" />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default NavQuery;
