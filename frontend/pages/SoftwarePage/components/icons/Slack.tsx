import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const Slack = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <path fill="#fff" d="M0 0h32v32H0z" />
      <g clipPath={`url(#${clipPathId})`} fillRule="evenodd" clipRule="evenodd">
        <path
          d="M13.066 5c-1.216 0-2.2.988-2.2 2.204 0 1.216.985 2.203 2.201 2.204h2.2V7.205A2.204 2.204 0 0 0 13.067 5Zm0 5.878H7.2A2.202 2.202 0 0 0 5 13.082a2.202 2.202 0 0 0 2.2 2.205h5.866a2.202 2.202 0 0 0 2.2-2.204 2.202 2.202 0 0 0-2.2-2.205Z"
          fill="#36C5F0"
        />
        <path
          d="M27 13.082a2.202 2.202 0 0 0-2.2-2.204 2.202 2.202 0 0 0-2.2 2.204v2.205h2.2a2.202 2.202 0 0 0 2.2-2.205Zm-5.867 0V7.204A2.203 2.203 0 0 0 18.933 5a2.202 2.202 0 0 0-2.2 2.204v5.878a2.202 2.202 0 0 0 2.2 2.205 2.202 2.202 0 0 0 2.2-2.205Z"
          fill="#2EB67D"
        />
        <path
          d="M18.933 27.044a2.202 2.202 0 0 0 2.2-2.204 2.202 2.202 0 0 0-2.2-2.204h-2.2v2.204c-.001 1.215.984 2.202 2.2 2.204Zm0-5.88H24.8a2.202 2.202 0 0 0 2.2-2.203 2.202 2.202 0 0 0-2.2-2.205h-5.866a2.202 2.202 0 0 0-2.2 2.204c-.001 1.217.983 2.204 2.199 2.205Z"
          fill="#ECB22E"
        />
        <path
          d="M5 18.96c0 1.217.984 2.204 2.2 2.205a2.202 2.202 0 0 0 2.2-2.204v-2.204H7.2A2.202 2.202 0 0 0 5 18.96Zm5.867 0v5.88a2.202 2.202 0 0 0 2.2 2.204 2.202 2.202 0 0 0 2.2-2.204v-5.877a2.2 2.2 0 1 0-4.4-.002Z"
          fill="#E01E5A"
        />
      </g>
      <defs>
        <clipPath id={clipPathId}>
          <path fill="#fff" transform="translate(5 5)" d="M0 0h22v22.044H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Slack;
