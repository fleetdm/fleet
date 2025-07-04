import React from "react";

import type { SVGProps } from "react";

const LastPass = (props: SVGProps<SVGSVGElement>) => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
    <g clipPath="url(#a)">
      <path fill="#CC0C38" d="M32 0H0v32h32V0Z" />
      <path
        fill="#fff"
        d="M8.07 18.087a1.809 1.809 0 1 0 0-3.617 1.809 1.809 0 0 0 0 3.617ZM13.983 18.087a1.809 1.809 0 1 0 0-3.617 1.809 1.809 0 0 0 0 3.617ZM19.896 18.087a1.809 1.809 0 1 0 0-3.617 1.809 1.809 0 0 0 0 3.617ZM25.67 11.757a.487.487 0 0 0-.974 0v8.417a.487.487 0 0 0 .974 0v-8.417Z"
      />
    </g>
    <defs>
      <clipPath id="a">
        <path fill="#fff" d="M0 0h32v32H0z" />
      </clipPath>
    </defs>
  </svg>
);
export default LastPass;
