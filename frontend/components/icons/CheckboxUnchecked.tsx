import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface ICheckboxUncheckedProps {
  color?: Colors;
}

const CheckboxUnchecked = ({
  color = "ui-fleet-black-25",
}: ICheckboxUncheckedProps) => {
  return (
    <svg width="16" height="17" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect
        className="checkbox-unchecked-state"
        x="1"
        y="1.5"
        width="14"
        height="14"
        rx="3"
        fill="#fff"
        stroke={COLORS[color]}
        strokeWidth="2"
      />
    </svg>
  );
};

export default CheckboxUnchecked;
