import React from "react";
import type { SVGProps } from "react";

const WindowsOS = (props: SVGProps<SVGSVGElement>) => {
  // 24px tiles use the 16px drawing at its native size.
  if (props.width === "24") {
    return (
      <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
        <path fill="#F9FAFC" d="M0 0h32v32H0z" />
        <g transform="translate(5.3333 5.3333) scale(1.3333)">
          <path
            d="M14.4219 8.28613V15.1416L6.94141 13.6592V8.28613H14.4219Z"
            fill="#515774"
          />
          <path
            d="M6.2041 8.28613V13.5166L1.57031 12.5908V8.28613H6.2041Z"
            fill="#515774"
          />
          <path
            d="M6.2041 7.70117H1.57031V3.42578L6.2041 2.49805V7.70117Z"
            fill="#515774"
          />
          <path
            d="M14.4229 7.70117H6.94141V2.35645L14.4229 0.859375V7.70117Z"
            fill="#515774"
          />
        </g>
      </svg>
    );
  }
  return (
    <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path fill="#F9FAFC" d="M0 0h32v32H0z" />
      <g transform="translate(6.6667 6.6667) scale(0.5833)">
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
};

export default WindowsOS;
