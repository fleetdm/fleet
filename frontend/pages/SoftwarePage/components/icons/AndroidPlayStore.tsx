import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const AndroidPlayStore = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <path fill="#e2e4ea" d="M0 0h32v32H0z" />
      <g clipPath={`url(#${clipPathId})`}>
        <path
          fill="#ea4335"
          d="M15.81 15.482 5.954 25.786a2.69 2.69 0 0 0 2.56 1.92q.768 0 1.344-.384l11.136-6.336z"
        />
        <path
          fill="#fbbc04"
          d="m25.794 13.69-4.8-2.752-5.376 4.736 5.44 5.312 4.8-2.688c.832-.448 1.408-1.344 1.408-2.304a2.87 2.87 0 0 0-1.472-2.304"
        />
        <path
          fill="#4285f4"
          d="M5.953 6.202c-.064.192-.064.448-.064.704v18.24c0 .256 0 .448.064.704l10.24-10.048z"
        />
        <path
          fill="#34a853"
          d="m15.874 15.994 5.12-5.056L9.922 4.666c-.384-.256-.896-.384-1.408-.384-1.216 0-2.304.832-2.56 1.92z"
        />
      </g>
      <defs>
        <clipPath id={clipPathId}>
          <path fill="#fff" d="M3.202 3.194h25.6v25.6h-25.6z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default AndroidPlayStore;
