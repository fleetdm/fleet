import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ISuccessProps {
  color?: Colors;
}

const Success = ({ color = "status-success" }: ISuccessProps) => {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M0 8c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8-8 3.58-8 8Zm11.29-2.71a1.003 1.003 0 0 1 1.42 1.42l-5 5c-.18.18-.43.29-.71.29-.28 0-.53-.11-.71-.29l-3-3a1.003 1.003 0 0 1 1.42-1.42L7 9.59l4.29-4.3Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Success;
