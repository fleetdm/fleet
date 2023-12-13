import React from "react";

import type { SVGProps } from "react";

const Chrome = (props: SVGProps<SVGSVGElement>) => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
    <path
      fill="#fff"
      stroke="#E2E4EA"
      d="M.5 8A7.5 7.5 0 0 1 8 .5h16A7.5 7.5 0 0 1 31.5 8v16a7.5 7.5 0 0 1-7.5 7.5H8A7.5 7.5 0 0 1 .5 24z"
    />
    <g clipPath="url(#Name=chrome_svg__a)">
      <path fill="#fff" d="M16 21.997a6 6 0 1 0 0-12 6 6 0 0 0 0 12" />
      <path
        fill="url(#Name=chrome_svg__b)"
        d="M16 10h10.39a11.997 11.997 0 0 0-20.781.002L10.804 19l.005-.001A5.992 5.992 0 0 1 16 10"
      />
      <path
        fill="#1A73E8"
        d="M16 20.75a4.75 4.75 0 1 0 0-9.5 4.75 4.75 0 0 0 0 9.5"
      />
      <path
        fill="url(#Name=chrome_svg__c)"
        d="M21.196 19.002 16 28a11.997 11.997 0 0 0 10.39-17.998H16l-.002.004a5.993 5.993 0 0 1 5.198 8.996"
      />
      <path
        fill="url(#Name=chrome_svg__d)"
        d="M10.804 19.002 5.61 10.003A11.997 11.997 0 0 0 16.001 28l5.195-8.998-.003-.004a5.992 5.992 0 0 1-10.389.004"
      />
    </g>
    <defs>
      <linearGradient
        id="Name=chrome_svg__b"
        x1={5.609}
        x2={26.391}
        y1={11.5}
        y2={11.5}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#D93025" />
        <stop offset={1} stopColor="#EA4335" />
      </linearGradient>
      <linearGradient
        id="Name=chrome_svg__c"
        x1={14.361}
        x2={24.752}
        y1={27.84}
        y2={9.842}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#FCC934" />
        <stop offset={1} stopColor="#FBBC04" />
      </linearGradient>
      <linearGradient
        id="Name=chrome_svg__d"
        x1={17.299}
        x2={6.908}
        y1={27.251}
        y2={9.253}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#1E8E3E" />
        <stop offset={1} stopColor="#34A853" />
      </linearGradient>
      <clipPath id="Name=chrome_svg__a">
        <path fill="#fff" d="M4 4h24v24H4z" />
      </clipPath>
    </defs>
  </svg>
);
export default Chrome;
