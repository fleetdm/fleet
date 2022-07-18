import React from "react";

import ReactTooltip from "react-tooltip";

const renderDiskSpaceGraph = (
  baseClass: string,
  gigsDiskSpaceAvailable: number,
  percentDiskSpaceAvailable: number,
  id: string
): JSX.Element => {
  const diskSpaceIndicator = () => {
    switch (true) {
      case gigsDiskSpaceAvailable < 16:
        return "red";
      case gigsDiskSpaceAvailable < 32:
        return "yellow";
      default:
        return "green";
    }
  };

  const diskSpaceTooltip = (): string | undefined => {
    switch (true) {
      case gigsDiskSpaceAvailable < 16:
        return "Not enough disk space available to install most small operating systems updates.";
      case gigsDiskSpaceAvailable < 32:
        return "Not enough disk space available to install most large operating systems updates.";
      default:
        return "Enough disk space available to install most operating systems updates.";
    }
  };
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
      <ReactTooltip
        className={"disk-space-tooltip"}
        place="bottom"
        type="dark"
        effect="solid"
        id={id}
        backgroundColor="#3e4771"
      >
        <span className={`${baseClass}__tooltip-text`}>
          {diskSpaceTooltip()}
        </span>
      </ReactTooltip>
      {gigsDiskSpaceAvailable} GB available
    </span>
  );
};

export default renderDiskSpaceGraph;
