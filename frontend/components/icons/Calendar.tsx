import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICalendarProps {
  color?: Colors;
  size?: IconSizes;
}

const Calendar = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: ICalendarProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M11.25 0C11.664 0 12 0.335 12 0.75V1.5H13C14.38 1.5 15.5 2.619 15.5 4V13.5C15.5 14.88 14.38 16 13 16H3C1.619 16 0.5 14.88 0.5 13.5V4C0.5 2.619 1.619 1.5 3 1.5H4V0.75C4 0.335 4.335 0 4.75 0C5.164 0 5.5 0.335 5.5 0.75V1.5H10.5V0.75C10.5 0.335 10.835 0 11.25 0ZM2 13.5C2 14.052 2.447 14.5 3 14.5H13C13.552 14.5 14 14.052 14 13.5V7.5H2V13.5ZM3 3C2.447 3 2 3.447 2 4V6H14V4C14 3.447 13.552 3 13 3H12V4.25C12 4.664 11.664 5 11.25 5C10.835 5 10.5 4.664 10.5 4.25V3H5.5V4.25C5.5 4.664 5.164 5 4.75 5C4.335 5 4 4.664 4 4.25V3H3Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Calendar;
