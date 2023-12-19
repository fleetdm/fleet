import React from "react";

import type { SVGProps } from "react";

const Word = (props: SVGProps<SVGSVGElement>) => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" {...props}>
    <path
      fill="#fff"
      stroke="#E2E4EA"
      d="M.5 8A7.5 7.5 0 0 1 8 .5h16A7.5 7.5 0 0 1 31.5 8v16a7.5 7.5 0 0 1-7.5 7.5H8A7.5 7.5 0 0 1 .5 24z"
    />
    <path
      fill="url(#Name=word_svg__a)"
      d="M24.625 6.5h-13.75c-.76 0-1.375.616-1.375 1.375v16.5c0 .76.616 1.375 1.375 1.375h13.75c.76 0 1.375-.616 1.375-1.375v-16.5c0-.76-.616-1.375-1.375-1.375"
    />
    <path
      fill="url(#Name=word_svg__b)"
      d="M9.5 20.938H26v3.437c0 .76-.616 1.375-1.375 1.375h-13.75c-.76 0-1.375-.616-1.375-1.375z"
    />
    <path fill="url(#Name=word_svg__c)" d="M26 16.125H9.5v4.813H26z" />
    <path fill="url(#Name=word_svg__d)" d="M26 11.313H9.5v4.812H26z" />
    <path
      fill="#000"
      fillOpacity={0.3}
      d="M9.5 13.375c0-1.14.923-2.062 2.063-2.062h4.124c1.14 0 2.063.923 2.063 2.062v8.25c0 1.14-.923 2.063-2.062 2.063H9.5z"
    />
    <path
      fill="url(#Name=word_svg__e)"
      d="M15 9.938H5.375c-.76 0-1.375.615-1.375 1.374v9.626c0 .759.616 1.375 1.375 1.375H15c.76 0 1.375-.616 1.375-1.375v-9.625c0-.76-.616-1.376-1.375-1.376"
    />
    <path
      fill="#fff"
      d="M14.313 12.697h-1.34l-1.051 4.486-1.15-4.495H9.639L8.48 17.183l-1.043-4.486H6.063l1.789 6.866h1.186l1.15-4.34 1.15 4.34h1.187z"
    />
    <defs>
      <linearGradient
        id="Name=word_svg__a"
        x1={9.5}
        x2={26}
        y1={9.708}
        y2={9.708}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#2B78B1" />
        <stop offset={1} stopColor="#338ACD" />
      </linearGradient>
      <linearGradient
        id="Name=word_svg__b"
        x1={9.5}
        x2={26}
        y1={23.945}
        y2={23.945}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#1B366F" />
        <stop offset={1} stopColor="#2657B0" />
      </linearGradient>
      <linearGradient
        id="Name=word_svg__c"
        x1={16.719}
        x2={26}
        y1={18.875}
        y2={18.875}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#20478B" />
        <stop offset={1} stopColor="#2D6FD1" />
      </linearGradient>
      <linearGradient
        id="Name=word_svg__d"
        x1={16.719}
        x2={26}
        y1={14.063}
        y2={14.063}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#215295" />
        <stop offset={1} stopColor="#2E84D3" />
      </linearGradient>
      <linearGradient
        id="Name=word_svg__e"
        x1={4}
        x2={17.063}
        y1={16.813}
        y2={16.813}
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="#223E74" />
        <stop offset={1} stopColor="#215091" />
      </linearGradient>
    </defs>
  </svg>
);
export default Word;
