import React from "react";
import { uniqueId } from "lodash";
import type { SVGProps } from "react";

const Word = (props: SVGProps<SVGSVGElement>) => {
  // Create unique IDs for the SVG gradients
  const gradAId = uniqueId("word-gradient-a-");
  const gradBId = uniqueId("word-gradient-b-");
  const gradCId = uniqueId("word-gradient-c-");
  const gradDId = uniqueId("word-gradient-d-");
  const gradEId = uniqueId("word-gradient-e-");

  return (
    <svg fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path fill="#fff" d="M0 0h32v32H0z" />
      <path
        d="M24.625 6.5h-13.75c-.76 0-1.375.616-1.375 1.375v16.5c0 .76.616 1.375 1.375 1.375h13.75c.76 0 1.375-.616 1.375-1.375v-16.5c0-.76-.616-1.375-1.375-1.375Z"
        fill={`url(#${gradAId})`}
      />
      <path
        d="M9.5 20.938H26v3.437c0 .76-.616 1.375-1.375 1.375h-13.75c-.76 0-1.375-.616-1.375-1.375v-3.438Z"
        fill={`url(#${gradBId})`}
      />
      <path d="M26 16.125H9.5v4.813H26v-4.813Z" fill={`url(#${gradCId})`} />
      <path d="M26 11.313H9.5v4.812H26v-4.813Z" fill={`url(#${gradDId})`} />
      <path
        d="M9.5 13.375c0-1.14.923-2.063 2.063-2.063h4.124c1.14 0 2.063.924 2.063 2.063v8.25c0 1.14-.923 2.063-2.063 2.063H9.5V13.374Z"
        fill="#000"
        fillOpacity=".3"
      />
      <path
        d="M15 9.938H5.375c-.76 0-1.375.615-1.375 1.374v9.626c0 .759.616 1.375 1.375 1.375H15c.76 0 1.375-.616 1.375-1.375v-9.625c0-.76-.616-1.376-1.375-1.376Z"
        fill={`url(#${gradEId})`}
      />
      <path
        d="M14.313 12.697h-1.34l-1.051 4.486-1.15-4.495H9.639L8.48 17.183l-1.043-4.486H6.063l1.789 6.866h1.186l1.15-4.34 1.15 4.34h1.187l1.789-6.866Z"
        fill="#fff"
      />
      <defs>
        <linearGradient
          id={gradAId}
          x1="9.5"
          y1="9.708"
          x2="26"
          y2="9.708"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#2B78B1" />
          <stop offset="1" stopColor="#338ACD" />
        </linearGradient>
        <linearGradient
          id={gradBId}
          x1="9.5"
          y1="23.945"
          x2="26"
          y2="23.945"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#1B366F" />
          <stop offset="1" stopColor="#2657B0" />
        </linearGradient>
        <linearGradient
          id={gradCId}
          x1="16.719"
          y1="18.875"
          x2="26"
          y2="18.875"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#20478B" />
          <stop offset="1" stopColor="#2D6FD1" />
        </linearGradient>
        <linearGradient
          id={gradDId}
          x1="16.719"
          y1="14.063"
          x2="26"
          y2="14.063"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#215295" />
          <stop offset="1" stopColor="#2E84D3" />
        </linearGradient>
        <linearGradient
          id={gradEId}
          x1="4"
          y1="16.813"
          x2="17.063"
          y2="16.813"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#223E74" />
          <stop offset="1" stopColor="#215091" />
        </linearGradient>
      </defs>
    </svg>
  );
};

export default Word;
