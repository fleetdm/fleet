import React from "react";

interface IChevronDownProps {
  color?: string;
}

const ChevronDown = ({ color = "#192147" }: IChevronDownProps) => {
  return (
    <svg width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="m8.751 10.891 4.144-4.297a.385.385 0 0 0 0-.528l-.927-.957a.345.345 0 0 0-.502 0L8.5 8.189l-2.966-3.08a.345.345 0 0 0-.502 0l-.927.957a.385.385 0 0 0 0 .528l4.144 4.297c.14.145.363.145.502 0Z"
        fill={color}
      />
    </svg>
  );
};

export default ChevronDown;
