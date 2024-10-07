import React from "react";

import type { SVGProps } from "react";

const Figma = (props: SVGProps<SVGSVGElement>) => (
  <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
    <path fill="#333" d="M0 0h32v32H0z" />
    <g clipPath="url(#a)">
      <path
        d="M12.1 27.718c2.153 0 3.9-1.747 3.9-3.9v-3.9h-3.9a3.902 3.902 0 0 0-3.9 3.9c0 2.153 1.747 3.9 3.9 3.9Z"
        fill="#0ACF83"
      />
      <path
        d="M8.2 16.017c0-2.152 1.747-3.9 3.9-3.9H16v7.8h-3.9a3.902 3.902 0 0 1-3.9-3.9Z"
        fill="#A259FF"
      />
      <path
        d="M8.2 8.217c0-2.153 1.747-3.9 3.9-3.9H16v7.8h-3.9a3.902 3.902 0 0 1-3.9-3.9Z"
        fill="#F24E1E"
      />
      <path
        d="M16 4.317h3.9c2.153 0 3.9 1.747 3.9 3.9 0 2.153-1.747 3.9-3.9 3.9H16v-7.8Z"
        fill="#FF7262"
      />
      <path
        d="M23.8 16.017c0 2.153-1.747 3.9-3.9 3.9a3.902 3.902 0 0 1-3.9-3.9c0-2.152 1.747-3.9 3.9-3.9 2.153 0 3.9 1.748 3.9 3.9Z"
        fill="#1ABCFE"
      />
    </g>
    <defs>
      <clipPath id="a">
        <path
          fill="#fff"
          transform="translate(8.198 4.317)"
          d="M0 0h15.604v23.401H0z"
        />
      </clipPath>
    </defs>
  </svg>
);
export default Figma;
