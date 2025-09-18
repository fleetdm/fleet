import React from "react";

import { PlacesType } from "react-tooltip-5";

import { COLORS } from "styles/var/colors";

import ProgressBar from "components/ProgressBar";
import TooltipWrapper from "components/TooltipWrapper";

interface IDiskSpaceIndicatorProps {
  baseClass: string;
  gigsDiskSpaceAvailable: number | "---";
  percentDiskSpaceAvailable: number;
  platform: string;
  tooltipPosition?: PlacesType;
}

const DiskSpaceIndicator = ({
  baseClass,
  gigsDiskSpaceAvailable,
  percentDiskSpaceAvailable,
  platform,
  tooltipPosition = "top",
}: IDiskSpaceIndicatorProps): JSX.Element => {
  if (gigsDiskSpaceAvailable === 0 || gigsDiskSpaceAvailable === "---") {
    return <span className={`${baseClass}__data`}>No data available</span>;
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
    <span className="disk-space-indicator">
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
      {gigsDiskSpaceAvailable} GB{baseClass === "info-flex" && " available"}
    </span>
  );
};

export default DiskSpaceIndicator;
