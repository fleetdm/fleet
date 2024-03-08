import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IArrowInternalLink {
  color?: Colors;
  size?: IconSizes;
}

const ArrowInternalLink = ({
  color = "core-fleet-blue",
  size = "medium",
}: IArrowInternalLink) => {
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
        d="M10.7929 10.7929C10.4024 11.1834 10.4024 11.8166 10.7929 12.2071C11.1834 12.5976 11.8166 12.5976 12.2071 12.2071L15.7071 8.70714C15.8946 8.5196 16 8.26525 16 8.00003C16 7.73482 15.8946 7.48046 15.7071 7.29293L12.2071 3.7929C11.8166 3.40237 11.1834 3.40237 10.7929 3.79289C10.4024 4.18341 10.4024 4.81658 10.7929 5.2071L12.5858 7.00002L1.00001 7.00002C0.447723 7.00002 7.18771e-06 7.44773 7.23599e-06 8.00002C7.28428e-06 8.5523 0.447722 9.00002 1.00001 9.00002L12.5858 9.00002L10.7929 10.7929Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ArrowInternalLink;
