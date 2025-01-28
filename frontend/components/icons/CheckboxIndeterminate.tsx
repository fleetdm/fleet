import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ICheckboxIndeterminateProps {
  color?: Colors;
}

const CheckboxIndeterminate = ({
  color = "core-fleet-blue",
}: ICheckboxIndeterminateProps) => {
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
      <rect x="3" y="7.5" width="10" height="2" rx="1" fill="#fff" />
    </svg>
  );
};

export default CheckboxIndeterminate;
