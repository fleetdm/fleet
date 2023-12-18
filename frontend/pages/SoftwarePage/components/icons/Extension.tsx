import React from "react";

import type { SVGProps } from "react";

const Extension = (props: SVGProps<SVGSVGElement>) => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
    <path
      fill="#F9FAFC"
      stroke="#E2E4EA"
      d="M.5 8A7.5 7.5 0 0 1 8 .5h16A7.5 7.5 0 0 1 31.5 8v16a7.5 7.5 0 0 1-7.5 7.5H8A7.5 7.5 0 0 1 .5 24z"
    />
    <path
      fill="#515774"
      fillRule="evenodd"
      d="M8.586 7.586A2 2 0 0 1 10 7h3c.527 0 1.044.18 1.432.568.388.388.568.905.568 1.432v2h2V9a2 2 0 0 1 2-2h3c.527 0 1.044.18 1.432.568.388.388.568.905.568 1.432v2a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V13a2 2 0 0 1 2-2V9a2 2 0 0 1 .586-1.414M22 11V9h-3v2zm-4 2H8v10h16V13h-1zm-5-4v2h-3V9z"
      clipRule="evenodd"
    />
  </svg>
);
export default Extension;
