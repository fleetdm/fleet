import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IPolicies {
  color?: Colors;
}
const Policy = ({ color = "core-fleet-black" }: IPolicies) => {
  return (
    <svg
      width="14"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 14 16"
    >
      <g
        clipPath="url(#a)"
        fillRule="evenodd"
        clipRule="evenodd"
        fill={COLORS[color]}
      >
        <path d="m6.951 9.838 3.112-3.015a.89.89 0 0 0 .27-.634.876.876 0 0 0-.27-.634.91.91 0 0 0-.64-.258.925.925 0 0 0-.638.258L6.313 7.95l-1.1-1.065a.919.919 0 0 0-1.477.29.877.877 0 0 0 .2.979l1.737 1.683a.91.91 0 0 0 .639.258.925.925 0 0 0 .64-.258Z" />
        <path d="M13.041 2.357v.001L7.345.067a.925.925 0 0 0-.69 0l-6.09 2.45a.906.906 0 0 0-.409.325A.882.882 0 0 0 0 3.34v2.98c0 2.066.634 4.083 1.82 5.796a10.637 10.637 0 0 0 4.84 3.82.926.926 0 0 0 .68 0 10.637 10.637 0 0 0 4.84-3.82A10.162 10.162 0 0 0 14 6.322V3.34a.88.88 0 0 0-.156-.499.905.905 0 0 0-.408-.325l-.395-.16Zm-.86 1.583v2.382a8.4 8.4 0 0 1-1.438 4.692A8.805 8.805 0 0 1 7 14.139a8.804 8.804 0 0 1-3.743-3.125 8.4 8.4 0 0 1-1.439-4.692V3.94L7 1.854l5.182 2.086Z" />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h14v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Policy;
