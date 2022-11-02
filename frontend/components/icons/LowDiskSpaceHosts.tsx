import React from "react";

const LowDiskSpaceHosts = () => {
  return (
    <svg width="33" height="20" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M19.5 5.5h-15a3 3 0 0 0-3 3v5a3 3 0 0 0 3 3h8M5.536 9.5v3M8.536 9.5v3"
        stroke="#515774"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="m10.767 16.477 8.553-14.97c.768-1.343 2.705-1.343 3.473 0l8.554 14.97c.762 1.333-.201 2.992-1.737 2.992H12.503c-1.536 0-2.498-1.66-1.736-2.992Z"
        fill="#515774"
      />
      <path d="M21 7.5v4" stroke="#fff" strokeWidth="2" strokeLinecap="round" />
      <circle cx="21" cy="14.5" r="1" fill="#fff" />
    </svg>
  );
};

export default LowDiskSpaceHosts;
