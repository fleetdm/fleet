import React from "react";
import type { SVGProps } from "react";

const WindowsOS = (props: SVGProps<SVGSVGElement>) => (
  <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
    <path fill="#F9FAFC" d="M0 0h32v32H0z" />
    <g transform="translate(8 8) scale(0.5)">
      <path
        d="M28.8408 16.5605V30.2705L13.8809 27.3105V16.5605H28.8408Z"
        fill="#515774"
      />
      <path
        d="M12.4102 16.5605V27.0205L3.14062 25.1709V16.5605H12.4102Z"
        fill="#515774"
      />
      <path
        d="M12.4102 15.3906H3.14062V6.84082L12.4102 4.99121V15.3906Z"
        fill="#515774"
      />
      <path
        d="M28.8506 15.3906H13.8809V4.70117L28.8506 1.71094V15.3906Z"
        fill="#515774"
      />
    </g>
  </svg>
);
export default WindowsOS;
