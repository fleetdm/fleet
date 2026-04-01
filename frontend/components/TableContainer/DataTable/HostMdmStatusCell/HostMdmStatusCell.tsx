import React from "react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import Icon from "components/Icon";
import NotSupported from "components/NotSupported";
import TooltipWrapper from "components/TooltipWrapper";
import { IHost } from "interfaces/host";

const baseClass = "host-mdm-status-cell";

const HostMdmStatusCell = ({
  row: {
    original: { id, mdm, platform },
  },
  cell: { value },
}: {
  row: { original: IHost };
  cell: { value: string };
}): JSX.Element => {
  if (platform === "chrome") {
    return NotSupported;
  }

  if (!value) {
    return <span className={`${baseClass}`}>{DEFAULT_EMPTY_CELL_VALUE}</span>;
  }

  return (
    <span className={`${baseClass}`}>
      {value}
      {mdm?.dep_profile_error && (
        <TooltipWrapper
          tipContent={
            <span className="tooltip__tooltip-text">
              Fleet hit Apple&apos;s API rate limit when preparing the macOS
              Setup Assistant for this host. Fleet will try again every hour.
            </span>
          }
          position="top"
          underline={false}
          showArrow
          tipOffset={8}
        >
          <Icon name="error-outline" color="status-error" size="medium" />
        </TooltipWrapper>
      )}
    </span>
  );
};

export default HostMdmStatusCell;
