import React from "react";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  MDM_STATUS_TOOLTIP,
} from "utilities/constants";
import paths from "router/paths";
import Icon from "components/Icon";
import CustomLink from "components/CustomLink";
import NotSupported from "components/NotSupported";
import TooltipWrapper from "components/TooltipWrapper";
import { IHost } from "interfaces/host";
import {
  MDM_ENROLLMENT_STATUS_UI_MAP,
  MdmEnrollmentStatus,
} from "interfaces/mdm";
import { isChrome, isLinuxLike } from "interfaces/platform";

const baseClass = "host-mdm-status-cell";

const HostMdmStatusCell = ({
  row: {
    original: { id, mdm, platform },
  },
  cell: { value },
}: {
  row: { original: IHost };
  cell: { value: MdmEnrollmentStatus };
}): JSX.Element => {
  if (isChrome(platform) || isLinuxLike(platform)) {
    return NotSupported;
  }

  if (!value) {
    return <span className={`${baseClass}`}>{DEFAULT_EMPTY_CELL_VALUE}</span>;
  }

  const displayValue =
    MDM_ENROLLMENT_STATUS_UI_MAP[value]?.displayName ?? value;

  return (
    <span className={`${baseClass}`}>
      {!MDM_STATUS_TOOLTIP[value] ? (
        displayValue
      ) : (
        <TooltipWrapper
          className={`${baseClass}__tooltip`}
          tipContent={MDM_STATUS_TOOLTIP[value]}
        >
          {displayValue}
        </TooltipWrapper>
      )}
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
