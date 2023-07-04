import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { IconSizes, ICON_SIZES } from "styles/var/icon_sizes";

interface ITrashCanProps {
  color?: Colors;
  size?: IconSizes;
}

const TrashCan = ({
  color = "core-fleet-blue",
  size = "medium",
}: ITrashCanProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
    >
      <path
        d="M13.25 2H10.5v-.5A1.5 1.5 0 0 0 9 0H7a1.5 1.5 0 0 0-1.5 1.5V2H2.75c-.69 0-1.25.56-1.25 1.25v1a.5.5 0 0 0 .5.5h12a.5.5 0 0 0 .5-.5v-1c0-.69-.56-1.25-1.25-1.25ZM6.5 1.5A.5.5 0 0 1 7 1h2a.5.5 0 0 1 .5.5V2h-3v-.5ZM2.449 5.75c-.09 0-.16.075-.156.164l.412 8.657A1.498 1.498 0 0 0 4.203 16h7.593c.802 0 1.46-.627 1.498-1.429l.413-8.657a.156.156 0 0 0-.156-.164H2.449ZM9.999 7a.5.5 0 1 1 1 0v6.5a.5.5 0 1 1-1 0V7ZM7.5 7a.5.5 0 1 1 1 0v6.5a.5.5 0 1 1-1 0V7ZM5 7a.5.5 0 1 1 1 0v6.5a.5.5 0 1 1-1 0V7Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default TrashCan;
