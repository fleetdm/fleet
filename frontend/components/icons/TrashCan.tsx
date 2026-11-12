import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ITrashCanProps {
  color?: Colors;
  size?: IconSizes;
}

const TrashCan = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: ITrashCanProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M6.5 5.5C6.914 5.5 7.25 5.835 7.25 6.25V11.75C7.25 12.164 6.914 12.5 6.5 12.5C6.085 12.5 5.75 12.164 5.75 11.75V6.25C5.75 5.835 6.085 5.5 6.5 5.5Z"
        fill={COLORS[color]}
      />
      <path
        d="M9.5 5.5C9.914 5.5 10.25 5.835 10.25 6.25V11.75C10.25 12.164 9.914 12.5 9.5 12.5C9.085 12.5 8.75 12.164 8.75 11.75V6.25C8.75 5.835 9.085 5.5 9.5 5.5Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.25 0C10.216 0 11 0.783 11 1.75V2H14.75C15.164 2 15.5 2.335 15.5 2.75C15.5 3.164 15.164 3.5 14.75 3.5H13.5V13.75C13.5 14.992 12.492 16 11.25 16H4.75C3.507 16 2.5 14.992 2.5 13.75V3.5H1.25C0.835 3.5 0.5 3.164 0.5 2.75C0.5 2.335 0.835 2 1.25 2H5V1.75C5 0.783 5.783 0 6.75 0H9.25ZM4 13.75C4 14.164 4.335 14.5 4.75 14.5H11.25C11.664 14.5 12 14.164 12 13.75V3.5H4V13.75ZM6.75 1.5C6.611 1.5 6.5 1.611 6.5 1.75V2H9.5V1.75C9.5 1.611 9.388 1.5 9.25 1.5H6.75Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default TrashCan;
