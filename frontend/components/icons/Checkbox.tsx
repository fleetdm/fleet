import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ICheckboxProps {
  color?: Colors;
}

const Checkbox = ({ color = "core-fleet-blue" }: ICheckboxProps) => {
  return (
    <svg width="16" height="17" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect
        className="checkbox-state"
        x="1"
        y="1.5"
        width="14"
        height="14"
        rx="3"
        fill={COLORS[color]}
        stroke={COLORS[color]}
        strokeWidth="2"
      />
      <g clipPath="url(#checkbox)">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M3.933 8.647c-.016 0-.033 0-.066.016a.828.828 0 0 1-.184-.066c.05-.033.134-.017.25.05Zm8.448-4.482c-.434-.233-.917.216-1.2.483-.65.633-1.2 1.366-1.816 2.032-.683.734-1.316 1.467-2.016 2.183-.4.4-.833.833-1.1 1.334-.6-.584-1.116-1.217-1.782-1.733-.483-.367-1.283-.634-1.267.25.034 1.149 1.05 2.382 1.8 3.165.316.334.733.683 1.216.7.583.033 1.183-.667 1.533-1.05.616-.666 1.117-1.416 1.683-2.099.733-.9 1.483-1.783 2.199-2.7.45-.566 1.866-1.965.75-2.565Z"
          fill="#fff"
        />
      </g>
      <defs>
        <clipPath id="checkbox">
          <path
            fill="#fff"
            transform="translate(3.2 4.1)"
            d="M0 0h9.6v8.8H0z"
          />
        </clipPath>
      </defs>
    </svg>
  );
};

export default Checkbox;
