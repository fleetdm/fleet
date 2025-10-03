import React from "react";
import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

/** Classes used for <CustomLink/> :hover styling of icon */
const baseClass = "external-link-icon";
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
      className={baseClass}
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 10 10"
    >
      <rect
        className={`${baseClass}__outline`}
        width="8"
        height="8"
        x="1"
        y="1"
        fill="none"
        stroke={COLORS[color]}
        rx="2"
      />
      <path
        className={`${baseClass}__arrow`}
        stroke={COLORS[color]}
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M4 3.333h2.667m0 0V6m0-2.667L3.333 6.668"
      />
    </svg>
  );
};

export default ExternalLink;
