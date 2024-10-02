import React from "react";

import type { SVGProps } from "react";

const TeamViewer = (props: SVGProps<SVGSVGElement>) => (
  <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
    <g clipPath="url(#a)">
      <path
        d="M28.657 0H3.307C2.425.011.002.03.002.03s-.004 2.42 0 3.292V32h32V0h-3.344ZM15.99 29.231a13.407 13.407 0 0 1-9.383-3.916A13.054 13.054 0 0 1 2.773 16a13.054 13.054 0 0 1 3.833-9.315A13.407 13.407 0 0 1 15.99 2.77a13.407 13.407 0 0 1 9.388 3.913A13.055 13.055 0 0 1 29.214 16a13.054 13.054 0 0 1-3.837 9.318 13.407 13.407 0 0 1-9.388 3.913Z"
        fill="url(#b)"
      />
      <path
        d="m28.257 16-9.171-4.24.737 2.716H12.16l.737-2.717-9.172 4.244 9.178 4.24-.737-2.716h7.663l-.737 2.717 9.165-4.24"
        fill="url(#c)"
      />
    </g>
    <defs>
      <linearGradient
        id="b"
        x1="15.989"
        y1="32.091"
        x2="15.989"
        y2="-.01"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#096FD2" />
        <stop offset="1" stopColor="#0E8EE9" />
      </linearGradient>
      <linearGradient
        id="c"
        x1="12.906"
        y1="20.252"
        x2="12.903"
        y2="11.764"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#096FD2" />
        <stop offset="1" stopColor="#0E8EE9" />
      </linearGradient>
      <clipPath id="a">
        <path fill="#fff" transform="translate(.001)" d="M0 0h32v32H0z" />
      </clipPath>
    </defs>
  </svg>
);
export default TeamViewer;
