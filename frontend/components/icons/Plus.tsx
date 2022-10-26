import React from "react";

interface IPlusProps {
  color?: string;
}

const Plus = ({ color = "#6a67fe" }: IPlusProps) => {
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
        d="M9.01 2.722a1.01 1.01 0 0 0-2.02 0v3.924H3.064a1.01 1.01 0 1 0 0 2.021H6.99v3.925a1.01 1.01 0 0 0 2.022 0V8.667h3.924a1.01 1.01 0 0 0 0-2.02H9.01V2.721Z"
        fill={color}
      />
    </svg>
  );
};

export default Plus;
