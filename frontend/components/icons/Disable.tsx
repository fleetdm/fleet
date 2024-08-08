import React from "react";
import { COLORS, Colors } from "styles/var/colors";

interface IDisable {
  color?: Colors;
}

const Disable = ({ color = "ui-fleet-black-50" }: IDisable) => {
  return (
    <svg
      width="16"
      height="16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <circle cx="8" cy="8" r="7" stroke={COLORS[color]} strokeWidth="2" />
      <path
        fill={COLORS[color]}
        d="m12.243 2.343 1.414 1.415-9.9 9.899-1.413-1.414z"
      />
    </svg>
  );
};

export default Disable;
