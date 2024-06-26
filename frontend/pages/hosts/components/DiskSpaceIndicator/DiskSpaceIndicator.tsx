import React from "react";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

interface IDiskSpaceIndicatorProps {
  baseClass: string;
  gigsDiskSpaceAvailable: number | "---";
  percentDiskSpaceAvailable: number;
  id: string;
  platform: string;
  tooltipPosition?: "top" | "bottom";
}

const DiskSpaceIndicator = ({
  baseClass,
  gigsDiskSpaceAvailable,
  percentDiskSpaceAvailable,
  id,
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
        return "red";
      } else if (gigsDiskSpaceAvailable < 32) {
        return "yellow";
      }
    }
    return "green";
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

  return (
    <span className={`${baseClass}__data`}>
      <div
        className={`${baseClass}__disk-space-wrapper tooltip`}
        data-tip
        data-for={`tooltip-${id}`}
      >
        <div className={`${baseClass}__disk-space`}>
          <div
            className={`${baseClass}__disk-space--${getDiskSpaceIndicatorColor()}`}
            style={{
              width: `${percentDiskSpaceAvailable}%`,
            }}
            title="disk space indicator"
          />
        </div>
      </div>
      {diskSpaceTooltipText && (
        <ReactTooltip
          className="disk-space-tooltip"
          place={tooltipPosition}
          type="dark"
          effect="solid"
          id={`tooltip-${id}`}
          backgroundColor={COLORS["tooltip-bg"]}
        >
          <span
            className={`${baseClass}__tooltip-text`}
            title="disk space tooltip"
          >
            {diskSpaceTooltipText}
          </span>
        </ReactTooltip>
      )}
      {gigsDiskSpaceAvailable} GB{baseClass === "info-flex" && " available"}
    </span>
  );
};

export default DiskSpaceIndicator;
