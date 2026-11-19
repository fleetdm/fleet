import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IUpdateProps {
  color?: Colors;
  size?: IconSizes;
}

const Update = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IUpdateProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8 3.5C8.048 3.5 8.096 3.504 8.142 3.513L8.147 3.514C8.236 3.532 8.318 3.567 8.393 3.613C8.401 3.618 8.409 3.621 8.416 3.626L8.42 3.629C8.459 3.656 8.496 3.686 8.53 3.719L11.719 6.91C12.012 7.203 12.012 7.677 11.719 7.97C11.426 8.263 10.952 8.263 10.659 7.97L8.75 6.061V11.25C8.75 11.664 8.414 12 8 12C7.585 12 7.25 11.664 7.25 11.25V6.059L5.339 7.97C5.047 8.263 4.572 8.263 4.279 7.97C3.986 7.677 3.986 7.203 4.279 6.91L7.469 3.719L7.583 3.625C7.617 3.602 7.655 3.583 7.693 3.566C7.706 3.56 7.719 3.554 7.733 3.549C7.765 3.537 7.798 3.528 7.832 3.52C7.843 3.517 7.855 3.514 7.867 3.512C7.879 3.51 7.892 3.507 7.905 3.505C7.936 3.501 7.967 3.5 8 3.5Z"
        fill={COLORS[color]}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8 0C12.418 0 16 3.581 16 8C16 12.418 12.418 16 8 16C3.581 16 0 12.418 0 8C0 3.581 3.581 0 8 0ZM8 1.5C4.41 1.5 1.5 4.41 1.5 8C1.5 11.589 4.41 14.5 8 14.5C11.589 14.5 14.5 11.589 14.5 8C14.5 4.41 11.589 1.5 8 1.5Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Update;
