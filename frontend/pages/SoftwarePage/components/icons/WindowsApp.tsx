import React from "react";

import type { SVGProps } from "react";

const WindowsApp = (props: SVGProps<SVGSVGElement>) => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
    <path
      fill="#F9FAFC"
      stroke="#E2E4EA"
      d="M.5 8A7.5 7.5 0 0 1 8 .5h16A7.5 7.5 0 0 1 31.5 8v16a7.5 7.5 0 0 1-7.5 7.5H8A7.5 7.5 0 0 1 .5 24z"
    />
    <path
      fill="#0078D4"
      fillRule="evenodd"
      d="M14.522 15.58H25V6L14.522 8.096zm-1.033.001H7V9.593l6.49-1.297zm0 8.143L7 22.428V16.4h6.49zM25 26l-10.478-2.076V16.4H25z"
      clipRule="evenodd"
    />
  </svg>
);
export default WindowsApp;
