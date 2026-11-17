import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

/** Classes used for <CustomLink/> :hover styling of icon */
const baseClass = "external-link-icon";
interface IExternalLinkProps {
  color?: Colors;
  size?: IconSizes;
}

const ExternalLink = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IExternalLinkProps) => {
  // Small is separately drawn at 12px with a 1px stroke to match 12px text;
  // scaling the 16px artwork down would make it too heavy.
  if (size === "small") {
    return (
      <svg
        className={`${baseClass} ${baseClass}--small`}
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        stroke={COLORS[color]}
        strokeWidth="1"
        strokeLinecap="round"
        strokeLinejoin="round"
        xmlns="http://www.w3.org/2000/svg"
      >
        <rect x="1.5" y="1.5" width="9" height="9" rx="1.5" />
        {/* Inline fill beats the shared `.external-link-icon path` hover fill. */}
        <path d="M4.5 7.5L7.5 4.5M5.5 4.5H7.5V6.5" style={{ fill: "none" }} />
      </svg>
    );
  }

  return (
    <svg
      className={baseClass}
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        className={`${baseClass}__arrow`}
        d="M9.741 5.509C10.155 5.509 10.491 5.845 10.491 6.259V8.967C10.491 9.381 10.155 9.716 9.741 9.717C9.327 9.717 8.991 9.381 8.991 8.967V8.069L6.789 10.271C6.496 10.564 6.021 10.564 5.729 10.271C5.436 9.979 5.436 9.504 5.729 9.211L7.931 7.009H7.032C6.618 7.009 6.282 6.673 6.282 6.259C6.282 5.845 6.618 5.509 7.032 5.509H9.741Z"
        fill={COLORS[color]}
      />
      <path
        className={`${baseClass}__outline`}
        fillRule="evenodd"
        clipRule="evenodd"
        d="M11.25 2.5C12.492 2.5 13.5 3.508 13.5 4.75V11.25C13.5 12.492 12.492 13.5 11.25 13.5H4.75C3.508 13.5 2.5 12.492 2.5 11.25V4.75C2.5 3.508 3.508 2.5 4.75 2.5H11.25ZM4.75 4C4.336 4 4 4.336 4 4.75V11.25C4 11.664 4.336 12 4.75 12H11.25C11.664 12 12 11.664 12 11.25V4.75C12 4.336 11.664 4 11.25 4H4.75Z"
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default ExternalLink;
