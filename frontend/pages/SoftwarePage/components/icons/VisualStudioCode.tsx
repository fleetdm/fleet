import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const VisualStudioCode = (props: SVGProps<SVGSVGElement>) => {
  const clipPathId = uniqueId("clip-path-");
  return (
    <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path fill="#fff" d="M0 0h32v32H0z" />
      <g clipPath={`url(#${clipPathId})`}>
        <path
          d="M5.64 12.864s-.53-.382.106-.892l1.483-1.326s.424-.447.873-.058l13.683 10.36v4.967s-.007.78-1.008.694L5.64 12.864Z"
          fill="#2489CA"
        />
        <path
          d="M9.167 16.066 5.64 19.273s-.362.27 0 .75l1.638 1.49s.389.418.963-.057l3.739-2.835-2.813-2.555Z"
          fill="#1070B3"
        />
        <path
          d="m15.359 16.093 6.467-4.939-.041-4.94s-.277-1.08-1.198-.518L11.98 13.53l3.379 2.563Z"
          fill="#0877B9"
        />
        <path
          d="M20.777 26.616c.375.384.83.258.83.258l5.041-2.484c.645-.44.555-.985.555-.985V8.573c0-.652-.668-.877-.668-.877L22.167 5.59c-.955-.59-1.58.106-1.58.106s.804-.579 1.198.517v19.611a.89.89 0 0 1-.087.387c-.115.232-.364.449-.963.358l.042.047Z"
          fill="#3C99D4"
        />
      </g>
      <defs>
        <clipPath id={clipPathId}>
          <path fill="#fff" transform="translate(5 5)" d="M0 0h22.403v22H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};
export default VisualStudioCode;
