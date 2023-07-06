import React from "react";

import { COLORS, Colors } from "styles/var/colors";

interface ICheckProps {
  color?: Colors;
}

const PendingPartial = ({ color = "ui-fleet-black-50" }: ICheckProps) => {
  return (
    <svg width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="8" cy="8" r="7" stroke={COLORS[color]} strokeWidth="2" />
      <circle cx="4.667" cy="8" r="1" fill={COLORS[color]} />
      <circle cx="7.667" cy="8" r="1" fill={COLORS[color]} />
      <circle cx="10.666" cy="8" r="1" fill={COLORS[color]} />
    </svg>
  );
};

export default PendingPartial;
