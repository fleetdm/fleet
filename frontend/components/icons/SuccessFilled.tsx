import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ISuccessFilledProps {
  color?: Colors;
  size?: IconSizes;
}

const SuccessFilled = ({
  color = "status-success",
  size = "medium",
}: ISuccessFilledProps) => {
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
        d="M8 0C12.418 0 16 3.581 16 8C16 12.418 12.418 16 8 16C3.581 16 0 12.418 0 8C0 3.581 3.581 0 8 0ZM11.687 5.052C11.322 4.751 10.781 4.804 10.481 5.169L7.08 9.296L5.463 7.679C5.128 7.344 4.585 7.344 4.251 7.679C3.916 8.014 3.916 8.557 4.251 8.891L6.536 11.177C6.707 11.348 6.942 11.439 7.184 11.427C7.425 11.415 7.651 11.302 7.804 11.116L11.804 6.258C12.105 5.893 12.052 5.353 11.687 5.052Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default SuccessFilled;
