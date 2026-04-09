import React from "react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import paths from "router/paths";
import Icon from "components/Icon";
import CustomLink from "components/CustomLink";
import NotSupported from "components/NotSupported";
import TooltipWrapper from "components/TooltipWrapper";
import { IHost } from "interfaces/host";

const baseClass = "host-mdm-status-cell";

const HostMdmStatusCell = ({
  row: {
    original: { mdm, platform },
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
              Migration or new Mac setup won&apos;t work. There&apos;s an issue
              with this host&apos;s Apple Business (AB) profile assignment.{" "}
              <CustomLink
                url={`${paths.HOST_DETAILS(id)}?show_mdm_status=true`}
                text="View details"
                variant="tooltip-link"
              />
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
