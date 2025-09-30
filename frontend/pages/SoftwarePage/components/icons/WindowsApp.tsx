import React from "react";

import type { SVGProps } from "react";


const WindowsApp = (props: SVGProps<SVGSVGElement>) => (
  <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
    <path fill="#F9FAFC" d="M0 0h32v32H0z" />
    <path
      fillRule="evenodd"
      clipRule="evenodd"
      d="M14.521 15.58H25V6L14.521 8.096v7.485Zm-1.032.001H7V9.593l6.49-1.297v7.285Zm0 8.143L7 22.427V16.4h6.49v7.325ZM25 26l-10.479-2.076V16.4H25V26Z"
      fill="#515774"
    />
  </svg>
);
export default WindowsApp;
