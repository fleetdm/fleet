import React from "react";

import type { SVGProps } from "react";

const WindowsAppRemote = (props: SVGProps<SVGSVGElement>) => {
  return (
    <svg
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 32 32"
      {...props}
    >
      <rect fill="#0078D4" width="32" height="32" />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M14.5215 15.5809H25V6L14.5215 8.09581V15.5809ZM13.4895 15.581H7V9.59294L13.4895 8.29554V15.581ZM13.4895 23.7245L7 22.4271V16.3992H13.4895V23.7245ZM25 26L14.5215 23.9242V16.3992H25V26Z"
        fill="white"
      />
    </svg>
  );
};
export default WindowsAppRemote;
