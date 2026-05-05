import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const ChromeApp = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  const fillPathBId = uniqueId("fill-path-b-");
  const fillPathCId = uniqueId("fill-path-c-");
  const fillPathDId = uniqueId("fill-path-d-");
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <path fill="#fff" d="M0 0h32v32H0z" />
      <g clipPath={`url(#${clipPathId})`}>
        <path d="M16 21.997a6 6 0 1 0 0-12 6 6 0 0 0 0 12Z" fill="#fff" />
        <path
          d="M16 10h10.39a11.997 11.997 0 0 0-20.781.002L10.804 19l.005-.001A5.992 5.992 0 0 1 16 10Z"
          fill={`url(#${fillPathBId})`}
        />
        <path
          d="M16 20.75a4.75 4.75 0 1 0 0-9.5 4.75 4.75 0 0 0 0 9.5Z"
          fill="#1A73E8"
        />
        <path
          d="M21.196 19.002 16 28a11.997 11.997 0 0 0 10.39-17.998H16l-.002.004a5.993 5.993 0 0 1 5.198 8.995Z"
          fill={`url(#${fillPathCId})`}
        />
        <path
          d="M10.804 19.002 5.61 10.003A11.997 11.997 0 0 0 16.001 28l5.195-8.998-.003-.004a5.992 5.992 0 0 1-10.389.004Z"
          fill={`url(#${fillPathDId})`}
        />
      </g>
      <defs>
        <linearGradient
          id={fillPathBId}
          x1="5.609"
          y1="11.5"
          x2="26.391"
          y2="11.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#D93025" />
          <stop offset="1" stopColor="#EA4335" />
        </linearGradient>
        <linearGradient
          id={fillPathCId}
          x1="14.361"
          y1="27.84"
          x2="24.752"
          y2="9.842"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FCC934" />
          <stop offset="1" stopColor="#FBBC04" />
        </linearGradient>
        <linearGradient
          id={fillPathDId}
          x1="17.299"
          y1="27.251"
          x2="6.908"
          y2="9.253"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#1E8E3E" />
          <stop offset="1" stopColor="#34A853" />
        </linearGradient>
        <clipPath id={clipPathId}>
          <path fill="#fff" transform="translate(4 4)" d="M0 0h24v24H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};
export default ChromeApp;
