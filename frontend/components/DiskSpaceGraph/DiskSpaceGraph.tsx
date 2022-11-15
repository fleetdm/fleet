import React from "react";

import ReactTooltip from "react-tooltip";

interface IDiskSpaceGraphProps {
  baseClass: string;
  gigsDiskSpaceAvailable: number | string;
  percentDiskSpaceAvailable: number;
  id: string;
  os: "mac" | "windows" | "linux";
}

const DiskSpaceGraph = ({
  baseClass,
  gigsDiskSpaceAvailable,
  percentDiskSpaceAvailable,
  id,
  os,
}: IDiskSpaceGraphProps): JSX.Element => {
  const diskSpaceIndicator = () => {
    switch (true) {
      case os === "linux":
        return "green";
      case gigsDiskSpaceAvailable < 16:
        return "red";
      case gigsDiskSpaceAvailable < 32:
        return "yellow";
      default:
        return "green";
    }
  };

  const diskSpaceTooltipText = ((): string | undefined => {
    switch (true) {
      case os === "linux":
        return undefined;
      case gigsDiskSpaceAvailable < 16:
        return "Not enough disk space available to install most small operating systems updates.";
      case gigsDiskSpaceAvailable < 32:
        return "Not enough disk space available to install most large operating systems updates.";
      default:
        return "Enough disk space available to install most operating systems updates.";
    }
  })();

  if (gigsDiskSpaceAvailable === 0 || gigsDiskSpaceAvailable === "---") {
    return <span className={`${baseClass}__data`}>No data available</span>;
  }

  // const tooltip = diskSpaceTooltipText ? (
  // ) : null;

  return (
    <span className={`${baseClass}__data`}>
      <div
        className={`${baseClass}__disk-space-wrapper tooltip`}
        data-tip
        data-for={id}
      >
        <div className={`${baseClass}__disk-space`}>
          <div
            className={`${baseClass}__disk-space--${diskSpaceIndicator()}`}
            style={{
              width: `${100 - percentDiskSpaceAvailable}%`,
            }}
          />
        </div>
      </div>
      {diskSpaceTooltipText && (
        <ReactTooltip
          className={"disk-space-tooltip"}
          place="bottom"
          type="dark"
          effect="solid"
          id={id}
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            {diskSpaceTooltipText}
          </span>
        </ReactTooltip>
      )}
      {gigsDiskSpaceAvailable} GB{baseClass === "info-flex" && " available"}
    </span>
  );
};

export default DiskSpaceGraph;
