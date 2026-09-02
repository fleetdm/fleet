import React from "react";

import { COLORS, Colors } from "styles/var/colors";

interface IABMIssueHostsProps {
  color?: Colors;
}

const ABMIssueHosts = ({
  color = "ui-fleet-black-75",
}: IABMIssueHostsProps) => {
  const fillColor = COLORS[color];
  const bgColor = COLORS["core-fleet-white"];

  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="36" height="24" fill="none">
      <path
        fill={fillColor}
        d="M7.918 1.845a1 1 0 1 0-2 0h2m-1 0h-1v3.044h2V1.845zM16.22 1.845a1 1 0 1 0-2 0h2m-1 0h-1v3.044h2V1.845z"
      />
      <path
        stroke={fillColor}
        strokeLinecap="round"
        strokeWidth="2"
        d="M6.918 1.66h8.302"
      />
      <path
        stroke={fillColor}
        strokeWidth="2"
        d="M18.754 5.073H3.384a2 2 0 0 0-2 2v11.681a2 2 0 0 0 2 2h15.37a2 2 0 0 0 2-2V7.074a2 2 0 0 0-2-2Z"
      />
      <path
        fill={fillColor}
        d="M9.934 12.473a1.5 1.5 0 0 0 3 0h-3m1.5-1.04h-1.5v1.04h3v-1.04z"
      />
      <path
        stroke={fillColor}
        strokeWidth="2"
        d="M1.384 10.147c1.076 0 4.427 1.153 9.224 1.153 4.796 0 8.763-1.153 10.146-1.153"
      />
      <path
        fill={fillColor}
        d="m16.191 21.278 7.783-13.622c.699-1.222 2.461-1.222 3.16 0l7.784 13.622C35.61 22.49 34.735 24 33.338 24H17.77c-1.398 0-2.273-1.51-1.58-2.722"
      />
      <path
        stroke={bgColor}
        strokeLinecap="round"
        strokeWidth="2"
        d="M25.503 13.109v3.64"
      />
      <path
        fill={bgColor}
        d="M25.503 20.388a.91.91 0 1 0 0-1.82.91.91 0 0 0 0 1.82"
      />
    </svg>
  );
};

export default ABMIssueHosts;
