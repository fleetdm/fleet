import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface INavHosts {
  color?: Colors;
}
const NavHosts = ({ color = "core-fleet-white" }: INavHosts) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <g clipPath="url(#a)" fill={COLORS[color]}>
        <path d="M14.2 0H1.8C.8 0 0 .8 0 1.8v3.6c0 1 .8 1.8 1.8 1.8h12.4c1 0 1.8-.8 1.8-1.8V1.8c0-1-.8-1.8-1.8-1.8Zm.6 5.4c0 .3-.3.6-.6.6H1.8c-.3 0-.6-.2-.6-.6V1.8c0-.3.3-.6.6-.6h12.4c.3 0 .6.3.6.6v3.6ZM14.2 8.8H1.8c-1 0-1.8.8-1.8 1.8v3.6c0 1 .8 1.8 1.8 1.8h12.4c1 0 1.8-.8 1.8-1.8v-3.6c0-1-.8-1.8-1.8-1.8Zm.6 5.4c0 .3-.3.6-.6.6H1.8c-.3 0-.6-.3-.6-.6v-3.6c0-.3.3-.6.6-.6h12.4c.3 0 .6.3.6.6v3.6Z" />
        <path d="M11.9 4.4a1 1 0 1 0 0-2 1 1 0 0 0 0 2ZM11.9 13.4a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default NavHosts;
