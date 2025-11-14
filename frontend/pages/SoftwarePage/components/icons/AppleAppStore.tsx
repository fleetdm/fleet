import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const AppleAppStore = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  const pathFillId = uniqueId("path-fill");
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <g clipPath={`url(#${clipPathId})`}>
        <path
          fill={`url(#${pathFillId})`}
          d="M25.536 0H6.464A6.46 6.46 0 0 0 0 6.464V25.54A6.46 6.46 0 0 0 6.464 32H25.54a6.464 6.464 0 0 0 6.464-6.464V6.464A6.467 6.467 0 0 0 25.536 0"
        />
        <path
          fill="#fff"
          d="m15.864 7.352.648-1.12a1.458 1.458 0 1 1 2.528 1.456l-6.244 10.808h4.516c1.464 0 2.284 1.72 1.648 2.912H5.72a1.45 1.45 0 0 1-1.456-1.456c0-.808.648-1.456 1.456-1.456h3.712l4.752-8.236L12.7 7.684a1.46 1.46 0 0 1 2.528-1.456zm-5.616 15.556-1.4 2.428A1.458 1.458 0 1 1 6.32 23.88l1.04-1.8c1.176-.364 2.132-.084 2.888.828m12.056-4.404h3.788c.808 0 1.456.648 1.456 1.456s-.648 1.456-1.456 1.456h-2.104l1.42 2.464a1.46 1.46 0 0 1-2.528 1.456c-2.392-4.148-4.188-7.252-5.38-9.32-1.22-2.104-.348-4.216.512-4.932.956 1.64 2.384 4.116 4.292 7.42"
        />
      </g>
      <defs>
        <linearGradient
          id={pathFillId}
          x1="16.002"
          x2="16.002"
          y1="0"
          y2="32"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#18bffb" />
          <stop offset="1" stopColor="#2072f3" />
        </linearGradient>
        <clipPath id={clipPathId}>
          <path fill="#fff" d="M0 0h32v32H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default AppleAppStore;
