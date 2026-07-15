import React from "react";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

export const getHostStatusTooltipText = (status: string): string => {
  if (status === "online") {
    return "Online hosts will respond to a live report.";
  }
  if (status === DEFAULT_EMPTY_CELL_VALUE) {
    return "Device is pending enrollment in Apple Business and status is not yet available.";
  }
  return "Offline hosts won't respond to a live report because they may be shut down, asleep, or not connected to the internet.";
};

export const getHostStatus = (
  status: string,
  mdmEnrollmentStatus?: string
): string => {
  if (mdmEnrollmentStatus === "Pending") {
    return DEFAULT_EMPTY_CELL_VALUE;
  }

  return status || DEFAULT_EMPTY_CELL_VALUE;
};

// getHardwareModelTooltip renders the tooltip shown behind an Apple device's
// marketing name, surfacing both the raw hardware model and the marketing name.
export const getHardwareModelTooltip = (
  hardwareModel: React.ReactNode,
  marketingName: React.ReactNode
): JSX.Element => (
  // Left-align to override the tooltip's default centered text.
  <div style={{ textAlign: "left" }}>
    <b>Model:</b> {hardwareModel}
    <br />
    <b>Marketing name:</b> {marketingName}
  </div>
);
