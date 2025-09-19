import React from "react";

import { PlacesType } from "react-tooltip-5";

import { COLORS } from "styles/var/colors";

import ProgressBar from "components/ProgressBar";
import TooltipWrapper from "components/TooltipWrapper";
import { isLinuxLike } from "interfaces/platform";

const baseClass = "disk-space-indicator";
interface IDiskSpaceIndicatorProps {
  gigsDiskSpaceAvailable: number | "---";
  percentDiskSpaceAvailable: number;
  gigsTotalDiskSpace?: number;
  gigsAllDiskSpace?: number;
  platform: string;
  inTableCell?: boolean;
  barTooltipPosition?: PlacesType;
  copyTooltipPosition?: PlacesType;
}

const DiskSpaceIndicator = ({
  gigsDiskSpaceAvailable,
  percentDiskSpaceAvailable,
  gigsTotalDiskSpace,
  gigsAllDiskSpace,
  platform,
  inTableCell = false,
  barTooltipPosition = "top",
  copyTooltipPosition = "bottom-end",
}: IDiskSpaceIndicatorProps): JSX.Element => {
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

  let barTooltip;
  if (platform === "darwin" || platform === "windows") {
    if (gigsDiskSpaceAvailable < 16) {
      barTooltip =
        "Not enough disk space available to install most small operating systems updates.";
    } else if (gigsDiskSpaceAvailable < 32) {
      barTooltip =
        "Not enough disk space available to install most large operating systems updates.";
    }
    barTooltip =
      "Enough disk space available to install most operating systems updates.";
  }

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

  let copyTooltip;
  if (isLinuxLike(platform) && !inTableCell) {
    copyTooltip = (
      <>
        System disk space: {gigsTotalDiskSpace} GB
        <br />
        {gigsAllDiskSpace ? <>All partitions: {gigsAllDiskSpace} GB</> : null}
      </>
    );
  }

  const renderCopy = () => (
    <>
      {gigsDiskSpaceAvailable} GB{!inTableCell && " available"}{" "}
    </>
  );
  return (
    <span className={baseClass}>
      {barTooltip ? (
        <TooltipWrapper
          position={barTooltipPosition}
          tipOffset={10}
          showArrow
          underline={false}
          tipContent={barTooltip}
        >
          {renderBar()}
        </TooltipWrapper>
      ) : (
        renderBar()
      )}
      {copyTooltip ? (
        <TooltipWrapper
          position={copyTooltipPosition}
          tipOffset={10}
          tipContent={copyTooltip}
          // fixedPositionStrategy
        >
          {renderCopy()}
        </TooltipWrapper>
      ) : (
        renderCopy()
      )}
    </span>
  );
};

export default DiskSpaceIndicator;
