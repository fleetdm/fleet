import React from "react";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";

interface IGraphCellProps {
  value: [number, number];
  customIdPrefix?: string;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const GraphCell = ({ value, customIdPrefix }: IGraphCellProps): JSX.Element => {
  const [gigs_disk_space_available, percent_disk_space_available] = value;

  const diskSpaceIndicator = () => {
    const diskSpaceAvailable = gigs_disk_space_available;
    switch (true) {
      case diskSpaceAvailable < 16:
        return "red";
      case diskSpaceAvailable < 32:
        return "yellow";
      default:
        return "green";
    }
  };

  const graphClassName = classnames(
    "data-table__graph",
    `data-table__graph--${generateClassTag(diskSpaceIndicator())}`
  );

  const diskSpaceTooltip = () => {
    const diskSpaceAvailable = gigs_disk_space_available;
    switch (true) {
      case diskSpaceAvailable < 16:
        return (
          <span className={"tooltip-text"}>
            Not enough disk space <br />
            available to install most <br />
            small operating systems <br />
            updates.
          </span>
        );
      case diskSpaceAvailable < 32:
        return (
          <span className={"tooltip-text"}>
            Not enough disk space <br />
            available to install most <br />
            large operating systems <br />
            updates.
          </span>
        );
      default:
        return (
          <span className={"tooltip-text"}>
            Enough disk space available <br />
            to install most operating <br />
            systems updates.
          </span>
        );
    }
  };

  if (gigs_disk_space_available > 0 || percent_disk_space_available > 0) {
    return (
      <span className="info-flex__data">
        <div className="info-flex__disk-space">
          <div
            className={`info-flex__disk-space-${diskSpaceIndicator()}`}
            style={{
              width: `${100 - percent_disk_space_available}%`,
            }}
            data-tip
            data-for={`${customIdPrefix || "disk-space"}__${customIdPrefix}`}
          />
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id={`${customIdPrefix || "disk-space"}__${customIdPrefix}`}
          backgroundColor="#3e4771"
        >
          {diskSpaceTooltip()}
        </ReactTooltip>
        {gigs_disk_space_available} GB available
      </span>
    );
  }
  return <span>No data available</span>;
};

export default GraphCell;
