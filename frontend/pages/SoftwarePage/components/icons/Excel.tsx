import React from "react";

import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const Excel = (props: SVGProps<SVGSVGElement>) => {
  const gradientId = uniqueId("gradient-");

  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
      <path fill="#fff" d="M0 0h32v32H0z" />
      <path
        d="M17.814 15.704 9.117 14.17v11.322a.937.937 0 0 0 .937.936H25.06a.936.936 0 0 0 .939-.936V21.32l-8.186-5.618Z"
        fill="#185C37"
      />
      <path
        d="M17.814 6h-7.76a.937.937 0 0 0-.937.936v4.171l8.697 5.107 4.605 1.532L26 16.214v-5.107L17.814 6Z"
        fill="#21A366"
      />
      <path d="M9.117 11.107h8.697v5.107H9.117v-5.107Z" fill="#107C41" />
      <path
        opacity=".1"
        d="M15.341 10.086H9.117v12.768h6.224a.943.943 0 0 0 .938-.936V11.022a.943.943 0 0 0-.938-.936Z"
        fill="#000"
      />
      <path
        opacity=".2"
        d="M14.83 10.596H9.116v12.768h5.712a.943.943 0 0 0 .939-.936V11.532a.943.943 0 0 0-.939-.936Z"
        fill="#000"
      />
      <path
        opacity=".2"
        d="M14.83 10.596H9.116v11.747h5.712a.943.943 0 0 0 .939-.936v-9.875a.943.943 0 0 0-.939-.936Z"
        fill="#000"
      />
      <path
        opacity=".2"
        d="M14.318 10.596H9.117v11.747h5.201a.943.943 0 0 0 .938-.936v-9.875a.943.943 0 0 0-.938-.936Z"
        fill="#000"
      />
      <path
        d="M4.938 10.596h9.38a.938.938 0 0 1 .938.936v9.364a.937.937 0 0 1-.938.936h-9.38A.935.935 0 0 1 4 20.896v-9.364a.936.936 0 0 1 .938-.936Z"
        fill={`url(#${gradientId})`}
      />
      <path
        d="m6.907 19.257 1.973-3.051-1.807-3.035h1.451l.986 1.943c.091.184.157.32.187.412h.014c.064-.148.132-.29.204-.429l1.054-1.923h1.336l-1.854 3.018 1.901 3.068h-1.421l-1.14-2.13a1.85 1.85 0 0 1-.134-.287h-.019c-.033.097-.077.19-.132.276l-1.173 2.138H6.907Z"
        fill="#fff"
      />
      <path
        d="M25.062 6h-7.248v5.107H26V6.936A.937.937 0 0 0 25.062 6Z"
        fill="#33C481"
      />
      <path d="M17.814 16.214H26v5.107h-8.186v-5.107Z" fill="#107C41" />
      <defs>
        <linearGradient
          id={gradientId}
          x1="5.96"
          y1="9.861"
          x2="13.297"
          y2="22.568"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#18884F" />
          <stop offset=".5" stopColor="#117E43" />
          <stop offset="1" stopColor="#0B6631" />
        </linearGradient>
      </defs>
    </svg>
  );
};

export default Excel;
