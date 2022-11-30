import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IDownload {
  color?: Colors;
}
const Download = ({ color = "core-fleet-blue" }: IDownload) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9 1a1 1 0 0 0-2 0v7.365L5.64 7.232a1 1 0 1 0-1.28 1.536l3 2.5a1 1 0 0 0 1.28 0l3-2.5a1 1 0 1 0-1.28-1.536L9 8.365V1ZM2 9a1 1 0 1 0-2 0v6a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V9a1 1 0 1 0-2 0v5H2V9Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Download;
