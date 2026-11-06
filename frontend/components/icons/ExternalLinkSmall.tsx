import React from "react";

import { COLORS, Colors } from "styles/var/colors";

/** Drawn at 12px with a 1px stroke to match 12px help text. Shares the
 * external-link-icon class so <CustomLink/> :hover styling applies. */
const baseClass = "external-link-icon";

interface IExternalLinkSmallProps {
  color?: Colors;
}

const ExternalLinkSmall = ({
  color = "ui-fleet-black-75",
}: IExternalLinkSmallProps) => {
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
};

export default ExternalLinkSmall;
