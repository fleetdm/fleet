import React from "react";
import { uniqueId } from "lodash";

const AppStore = () => {
  const pathFillId = uniqueId("path-fill");
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="42" height="42" fill="none">
      <path
        fill={`url(#${pathFillId})`}
        d="M33.516 0H8.484A8.48 8.48 0 0 0 0 8.484v25.037A8.48 8.48 0 0 0 8.484 42h25.037a8.484 8.484 0 0 0 8.484-8.484V8.484A8.49 8.49 0 0 0 33.516 0"
      />
      <path
        fill="#fff"
        d="m20.822 9.65.85-1.47a1.913 1.913 0 1 1 3.318 1.91l-8.195 14.186h5.927c1.922 0 2.998 2.258 2.163 3.822H7.508c-1.06 0-1.911-.85-1.911-1.911s.85-1.911 1.91-1.911h4.873l6.237-10.81-1.948-3.38a1.916 1.916 0 0 1 .703-2.615 1.916 1.916 0 0 1 2.615.703zM13.45 30.066l-1.838 3.187a1.913 1.913 0 1 1-3.318-1.911L9.66 28.98c1.544-.478 2.798-.11 3.79 1.087m15.823-5.78h4.972c1.06 0 1.91.85 1.91 1.91s-.85 1.912-1.91 1.912h-2.762l1.864 3.233a1.916 1.916 0 0 1-.703 2.615 1.916 1.916 0 0 1-2.615-.703c-3.14-5.445-5.497-9.519-7.061-12.233-1.601-2.762-.457-5.533.672-6.473 1.255 2.152 3.129 5.402 5.633 9.739"
      />
      <defs>
        <linearGradient
          id={pathFillId}
          x1="21.003"
          x2="21.003"
          y1="0"
          y2="42"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#18bffb" />
          <stop offset="1" stopColor="#2072f3" />
        </linearGradient>
      </defs>
    </svg>
  );
};

export default AppStore;
