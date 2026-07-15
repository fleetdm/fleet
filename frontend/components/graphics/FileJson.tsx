import React from "react";

import { uniqueId } from "lodash";

const FileJson = () => {
  const clipPathId = uniqueId("clip-path-");

  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="34" height="40" fill="none">
      <g clipPath={`url(#${clipPathId})`}>
        <path
          fill="#fff"
          stroke="#192147"
          strokeWidth={0.5}
          d="M29.333 39.75H4.667a2.417 2.417 0 0 1-2.417-2.416V2.667A2.417 2.417 0 0 1 4.667.25h19.562c.64 0 1.255.255 1.709.708l5.104 5.105c.453.453.708 1.068.708 1.709v29.562a2.417 2.417 0 0 1-2.417 2.416Z"
        />
        <path
          fill="#C5C7D1"
          d="M23.5.5h.834l.5 6.5 6.666.5v1h-6a2 2 0 0 1-2-2v-6Z"
        />
        <path
          stroke="#192147"
          strokeWidth={0.5}
          d="M24.5.334v5.667c0 .736.597 1.333 1.333 1.333h6"
        />
        <path
          fill="#C5C7D1"
          d="M2.5 20h25a2 2 0 0 1 2 2v13a2 2 0 0 1-2 2h-25V20Z"
        />
        <rect
          width={27.7}
          height={16.35}
          x={0.25}
          y={18.25}
          fill="#515774"
          rx={1.75}
        />
        <text
          x={14.1}
          y={26.7}
          fill="#fff"
          fontFamily="Inter, sans-serif"
          fontSize={7.5}
          fontWeight={700}
          textAnchor="middle"
        >
          json
        </text>
        <rect
          width={27.7}
          height={16.35}
          x={0.25}
          y={18.25}
          stroke="#192147"
          strokeWidth={0.5}
          rx={1.75}
        />
      </g>
      <defs>
        <clipPath id={clipPathId}>
          <path fill="#fff" d="M0 0h34v40H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default FileJson;
