import React from "react";

import { PlacesType } from "react-tooltip-5";
import NotSupported from "components/NotSupported";

import { COLORS } from "styles/var/colors";

import ProgressBar from "components/ProgressBar";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "disk-space-indicator";
interface IDiskSpaceIndicatorProps {
  gigsDiskSpaceAvailable: number | "---";
  percentDiskSpaceAvailable: number;
  gigsTotalDiskSpace?: number;
  gigsAllDiskSpace?: number;
  platform: string;
  inTableCell?: boolean;
  tooltipPosition?: PlacesType;
}

const DiskSpaceIndicator = ({
  gigsDiskSpaceAvailable,
  percentDiskSpaceAvailable,
  gigsTotalDiskSpace,
  gigsAllDiskSpace,
  platform,
  inTableCell = false,
  tooltipPosition = "top",
}: IDiskSpaceIndicatorProps): JSX.Element => {
  // Check if storage measurement is not supported (sentinel value -1)
  if (
    typeof gigsDiskSpaceAvailable === "number" &&
    gigsDiskSpaceAvailable < 0
  ) {
    return NotSupported;
  }

  if (gigsDiskSpaceAvailable === 0 || gigsDiskSpaceAvailable === "---") {
    return <span className={`${baseClass}__empty`}>No data available</span>;
  }

  const getDiskSpaceIndicatorColor = (): string => {
    // return space-dependent indicator colors for mac and windows hosts, green for linux
    if (platform === "darwin" || platform === "windows") {
      if (gigsDiskSpaceAvailable < 16) {
        return COLORS["ui-error"];
      } else if (gigsDiskSpaceAvailable < 32) {
        return COLORS["ui-warning"];
      }
    }
    return COLORS["status-success"];
  };

  const diskSpaceTooltipText = ((): string | undefined => {
    if (platform === "darwin" || platform === "windows") {
      if (gigsDiskSpaceAvailable < 16) {
        return "Not enough disk space available to install most small operating systems updates.";
      } else if (gigsDiskSpaceAvailable < 32) {
        return "Not enough disk space available to install most large operating systems updates.";
      }
      return "Enough disk space available to install most operating systems updates.";
    }
    return undefined;
  })();

  const renderBar = () => (
    <ProgressBar
      sections={[
        {
          color: getDiskSpaceIndicatorColor(),
          portion: percentDiskSpaceAvailable / 100,
        },
      ]}
      width="small"
    />
  );

  return (
    <span className={baseClass}>
      {diskSpaceTooltipText ? (
        <TooltipWrapper
          position={tooltipPosition}
          tipOffset={10}
          showArrow
          underline={false}
          tipContent={diskSpaceTooltipText}
        >
          {renderBar()}
        </TooltipWrapper>
      ) : (
        renderBar()
      )}
      {gigsDiskSpaceAvailable} GB{!inTableCell && " available"}
    </span>
  );
};

export default DiskSpaceIndicator;
