import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface ICentosProps {
  color?: Colors;
  size?: IconSizes;
}

const Centos = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: ICentosProps) => {
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
        d="M10.343 2.343H13.657V5.656L16 8L13.657 10.343V13.657H10.343L8 16L5.656 13.657H2.343V10.343L0 8L2.343 5.656V2.343H5.656L8 0L10.343 2.343ZM7.288 11.2V13.6H8.711V11.2L7.999 10.488L7.288 11.2ZM5.234 9.76L3.537 11.457L4.543 12.463L6.24 10.765V9.76H5.234ZM9.759 9.76V10.765L11.457 12.462L12.462 11.457L10.765 9.76H9.759ZM7.999 7.074L7.336 6.4L7.345 7.345L6.399 7.337L7.074 8L6.399 8.663L7.345 8.654L7.337 9.6L8 8.925L8.663 9.6L8.654 8.654L9.6 8.662L8.925 7.999L9.599 7.337L8.654 7.345L8.662 6.399L7.999 7.074ZM2.399 7.288V8.711H4.8L5.511 8L4.8 7.288H2.399ZM11.2 7.288L10.488 8L11.2 8.711H13.6V7.288H11.2ZM3.537 4.543L5.234 6.24H6.24V5.234L4.543 3.537L3.537 4.543ZM9.759 5.234V6.24H10.765L12.462 4.543L11.457 3.537L9.759 5.234ZM7.288 2.4V4.8L8 5.511L8.711 4.8V2.4H7.288Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Centos;
