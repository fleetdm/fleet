import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ICheckProps {
  color?: Colors;
}

const Check = ({ color = "core-fleet-blue" }: ICheckProps) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      className="check"
    >
      <g clipPath="url(#a)">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M2.917 8.684c-.02 0-.042 0-.083.02a1.035 1.035 0 0 1-.23-.083c.063-.041.167-.021.313.063Zm10.56-5.603c-.543-.292-1.147.27-1.5.604-.812.791-1.5 1.708-2.27 2.54-.855.917-1.646 1.834-2.52 2.73-.5.5-1.042 1.04-1.375 1.666-.75-.73-1.396-1.52-2.228-2.166C2.98 7.996 1.98 7.663 2 8.767c.042 1.437 1.313 2.978 2.25 3.957.395.417.916.854 1.52.874.73.042 1.479-.833 1.916-1.312.77-.832 1.396-1.77 2.104-2.623.916-1.125 1.854-2.23 2.748-3.374.563-.709 2.333-2.458.938-3.208Z"
          fill={COLORS[color]}
        />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="#fff" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Check;
