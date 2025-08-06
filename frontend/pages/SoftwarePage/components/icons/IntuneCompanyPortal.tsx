import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const IntuneCompanyPortal = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  const maskBId = uniqueId("mask-b-");
  const fillPathCId = uniqueId("fill-path-c-");
  const fillPathDId = uniqueId("fill-path-d-");

  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <g clipPath={`url(#${clipPathId})`}>
        <path fill="#fff" d="M32 0H0v32h32V0Z" />
        <mask
          id={maskBId}
          width="26"
          height="18"
          x="3"
          y="4"
          maskUnits="userSpaceOnUse"
          style={{ maskType: "luminance" }}
        >
          <path
            fill="#fff"
            d="M27.918 4.477H4.082a.395.395 0 0 0-.395.395v16.33c0 .218.177.395.395.395h23.836a.395.395 0 0 0 .395-.395V4.872a.395.395 0 0 0-.395-.395Z"
          />
        </mask>
        <g mask={`url(#${maskBId})`}>
          <path
            fill="#134583"
            d="M27.918 4.477H4.082a.395.395 0 0 0-.395.395v16.33c0 .218.177.395.395.395h23.836a.395.395 0 0 0 .395-.395V4.872a.395.395 0 0 0-.395-.395Z"
          />
          <path fill="#0862B2" d="M11.72 4.477H3.687v8.56h8.033v-8.56Z" />
          <path fill="#0B87DA" d="M11.72 4.477h8.099v8.56H11.72v-8.56Z" />
          <path fill="#31BCEF" d="M28.313 4.477h-8.494v8.56h8.494v-8.56Z" />
          <path fill="#1DA0E4" d="M28.313 17.185h-8.494v-4.148h8.494v4.148Z" />
          <path fill="#0680D7" d="M28.313 17.185h-8.494v4.939h8.494v-4.939Z" />
        </g>
        <path
          fill={`url(#${fillPathCId})`}
          d="M12.263 17.995a4.108 4.108 0 1 0 0-8.216 4.108 4.108 0 0 0 0 8.216Z"
        />
        <path
          fill={`url(#${fillPathDId})`}
          d="M5.376 27.528h13.61c.664-11.431-14.164-11.92-13.61 0Z"
        />
      </g>
      <defs>
        <linearGradient
          id={fillPathCId}
          x1="10.077"
          x2="14.353"
          y1="10.173"
          y2="17.58"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#4EE3FE" />
          <stop offset="1" stopColor="#21A6E5" />
        </linearGradient>
        <linearGradient
          id={fillPathDId}
          x1="8.553"
          x2="11.982"
          y1="19.19"
          y2="28.447"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#4EE3FE" />
          <stop offset="1" stopColor="#21A6E5" />
        </linearGradient>
        <clipPath id={clipPathId}>
          <path fill="#fff" d="M0 0h32v32H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};
export default IntuneCompanyPortal;
