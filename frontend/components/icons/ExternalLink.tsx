import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IExternalLinkProps {
  size: IconSizes;
  color: Colors;
}

const ExternalLink = ({
  size = "small",
  color = "ui-fleet-black-75",
}: IExternalLinkProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 10 10"
    >
      <rect width="8" height="8" x="1" y="1" stroke="#515774" rx="2" />
      <path
        stroke={COLORS[color]}
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M4 3.333h2.667m0 0V6m0-2.667L3.333 6.668"
      />
    </svg>
  );
};

export default ExternalLink;
