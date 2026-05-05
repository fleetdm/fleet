import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IUploadProps {
  color?: Colors;
}

const Upload = ({ color = "ui-fleet-black-75" }: IUploadProps) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="17"
      height="17"
      viewBox="0 0 17 17"
      fill="none"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M7.5 11C7.5 11.5523 7.94772 12 8.5 12C9.05228 12 9.5 11.5523 9.5 11V3.63504L10.8598 4.76822C11.2841 5.12179 11.9147 5.06446 12.2682 4.64018C12.6218 4.21591 12.5645 3.58534 12.1402 3.23178L9.14018 0.731779C8.76934 0.42274 8.23066 0.42274 7.85982 0.731779L4.85982 3.23178C4.43554 3.58534 4.37821 4.21591 4.73178 4.64018C5.08534 5.06446 5.71591 5.12179 6.14018 4.76822L7.5 3.63504L7.5 11ZM2.5 9.5C2.5 8.94771 2.05228 8.5 1.5 8.5C0.947715 8.5 0.5 8.94771 0.5 9.5L0.5 15.5C0.5 16.0523 0.947715 16.5 1.5 16.5H15.5C16.0523 16.5 16.5 16.0523 16.5 15.5V9.5C16.5 8.94771 16.0523 8.5 15.5 8.5C14.9477 8.5 14.5 8.94771 14.5 9.5V14.5H2.5L2.5 9.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Upload;
