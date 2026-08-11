import React from "react";

import { isAppleDevice } from "interfaces/platform";
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

// getHardwareModelDisplay computes how a host's hardware model is presented.
// Apple devices with a known marketing name (e.g. "MacBook Pro (16-inch,
// 2021)") show it in place of the raw model and reveal the raw model in a
// tooltip; otherwise the raw model is shown with no supplemental tooltip.
// An empty field may be "" (raw API data) or DEFAULT_EMPTY_CELL_VALUE (data
// that went through normalizeEmptyValues) — both count as "no value".
export const getHardwareModelDisplay = (
  platform: string,
  hardwareModel: string,
  hardwareMarketingName: string
): { value: string; tooltip?: JSX.Element; alwaysShowTooltip: boolean } => {
  const isEmpty = (val: string) => !val || val === DEFAULT_EMPTY_CELL_VALUE;

  const marketingName =
    isAppleDevice(platform) && !isEmpty(hardwareMarketingName)
      ? hardwareMarketingName
      : "";
  // Only reveal the raw model on hover when we're actually showing a distinct
  // marketing name in its place. When there's no mapping the marketing name is
  // empty (or echoes the raw model), so we show the raw model plainly with no
  // tooltip.
  const showModelTooltip =
    !!marketingName &&
    !isEmpty(hardwareModel) &&
    marketingName !== hardwareModel;

  return {
    value: marketingName || hardwareModel,
    tooltip: showModelTooltip ? (
      // Left-align to override the tooltip's default centered text.
      <div style={{ textAlign: "left" }}>
        <b>Model:</b> {hardwareModel}
        <br />
        <b>Marketing name:</b> {marketingName}
      </div>
    ) : undefined,
    alwaysShowTooltip: showModelTooltip,
  };
};
