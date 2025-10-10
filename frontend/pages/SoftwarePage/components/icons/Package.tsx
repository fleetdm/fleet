import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const Package = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <path fill="#fff" d="M0 0h32v32H0z" />
      <g clipPath={`url(#${clipPathId})`}>
        <path
          opacity=".75"
          d="m16.033 25.645 8.354-4.822L16.033 16 7.68 20.823l8.353 4.822Z"
          fill="#FF9701"
        />
        <path
          opacity=".75"
          d="M24.386 20.824v-9.68l-8.353-4.823v9.646l8.353 4.857Z"
          fill="#FFE400"
        />
        <path
          opacity=".75"
          d="m7.68 20.824 8.353-4.857V6.321L7.68 11.145v9.679Z"
          fill="#fff"
        />
        <path
          d="m16.033 15.967 8.354-4.822-8.354-4.824-8.353 4.824 8.353 4.822Z"
          fill="#D8A270"
        />
        <path
          d="m12.627 8.52-.22-.135-.273.441.221.136.272-.442Zm-.272.442 7.864 4.84.272-.442-7.864-4.84-.272.442Z"
          fill="#4D2D0B"
        />
        <path
          d="M7.652 11.161v9.68l8.352 4.822v-9.68l-8.352-4.822Z"
          fill="#9F6B3B"
        />
        <path
          d="M16.033 25.646v-9.68l8.353-4.822v9.68l-8.353 4.822Z"
          fill="#C7803F"
        />
        <path
          d="M16.033 25.646V16l8.353-4.856M7.68 11.144 16.033 16"
          stroke="#4D2D0B"
          strokeWidth=".519"
          strokeMiterlimit="10"
        />
        <path
          d="M7.68 11.144v9.68l8.353 4.821 8.354-4.822v-9.679l-8.354-4.79-8.353 4.79Z"
          stroke="#4D2D0B"
          strokeWidth=".519"
          strokeMiterlimit="10"
        />
        <path
          d="m17.33 20.84 2.42-1.21v2.42l-2.42 1.21v-2.42Z"
          fill="#DE9E64"
        />
      </g>
      <defs>
        <clipPath id={clipPathId}>
          <path fill="#fff" transform="translate(7 6)" d="M0 0h18v20H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};
export default Package;
