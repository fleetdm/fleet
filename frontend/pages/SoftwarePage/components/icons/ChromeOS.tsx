import React from "react";

import type { SVGProps } from "react";

const ChromeOS = (props: SVGProps<SVGSVGElement>) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 32 32"
    {...props}
  >
    <path fill="#F9FAFC" d="M0 0h32v32H0z" />
    <path
      d="M5 16c0-2.002.536-3.884 1.473-5.539l4.719 8.216A5.502 5.502 0 0 0 16 21.5a5.67 5.67 0 0 0 1.753-.284l-3.278 5.68C9.12 26.155 5 21.557 5 16Zm15.688 2.819A5.28 5.28 0 0 0 21.5 16a5.502 5.502 0 0 0-1.86-4.125h6.56c.516 1.272.8 2.668.8 4.125 0 6.076-4.924 10.961-11 11l4.688-8.181ZM25.53 10.5H16a5.473 5.473 0 0 0-5.393 4.413L7.328 9.23A10.977 10.977 0 0 1 16 5c4.073 0 7.627 2.212 9.53 5.5ZM12.219 16a3.781 3.781 0 1 1 7.562 0 3.781 3.781 0 0 1-7.562 0Z"
      fill="#515774"
    />
  </svg>
);
export default ChromeOS;
