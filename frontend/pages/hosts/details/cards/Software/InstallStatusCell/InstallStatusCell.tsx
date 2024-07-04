import React, { ReactNode } from "react";

import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";
import { dateAgo } from "utilities/date_format";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";

const baseClass = "install-status-cell";

type IStatusValue = SoftwareInstallStatus | "avaiableForInstall";

export type IStatusDisplayConfig = {
  iconName:
    | "success"
    | "pending-outline"
    | "error"
    | "install"
    | "install-self-service";
  displayText: string;
  tooltip: (args: {
    softwareName?: string | null;
    lastInstalledAt?: string;
  }) => ReactNode;
};

export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  IStatusValue | "selfService",
  IStatusDisplayConfig
> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ lastInstalledAt: lastInstall }) => (
      <>
        Fleet installed software on these hosts. (
        {dateAgo(lastInstall as string)})
      </>
    ),
  },
  pending: {
    iconName: "pending-outline",
    displayText: "Pending",
    tooltip: () => "Fleet will install software when the host comes online.",
  },
  failed: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ lastInstalledAt: lastInstall }) => (
      <>
        Fleet failed to install software ({dateAgo(lastInstall as string)} ago).
        Select <b>Actions &gt; Software details</b> to see more.
      </>
    ),
  },
  avaiableForInstall: {
    iconName: "install",
    displayText: "Available for install",
    tooltip: ({ softwareName }) => (
      <>
        {softwareName ? <b>{softwareName}</b> : "Software"} can be installed on
        the host. Select <b>Actions {">"} Install</b> to install.
      </>
    ),
  },
  selfService: {
    iconName: "install-self-service",
    displayText: "Self-service",
    tooltip: ({ softwareName }) => (
      <>
        {softwareName ? <b>{softwareName}</b> : "Software"} can be installed on
        the host. End users can install from{" "}
        <b>Fleet Desktop {">"} Self-service</b>.
      </>
    ),
  },
};

const InstallStatusCell = ({
  status,
  last_install,
  package_available_for_install: softwareName,
  self_service,
}: IHostSoftware) => {
  const lastInstalledAt = last_install?.installed_at;

  let displayStatus: keyof typeof INSTALL_STATUS_DISPLAY_OPTIONS;

  if (status !== null) {
    displayStatus = status;
  } else if (softwareName && self_service) {
    displayStatus = "selfService";
  } else if (softwareName) {
    displayStatus = "avaiableForInstall";
  } else {
    return <TextCell value="---" grey italic />;
  }

  const displayConfig = INSTALL_STATUS_DISPLAY_OPTIONS[displayStatus];
  const tooltipId = uniqueId();

  return (
    <div className={`${baseClass}__status-content`}>
      <div
        className={`${baseClass}__status-with-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        <Icon name={displayConfig.iconName} />{" "}
        <span>{displayConfig.displayText}</span>
      </div>
      <ReactTooltip
        className={`${baseClass}__status-tooltip`}
        effect="solid"
        backgroundColor="#3e4771"
        id={tooltipId}
        data-html
      >
        <span className={`${baseClass}__status-tooltip-text`}>
          {displayConfig.tooltip({
            softwareName,
            lastInstalledAt,
          })}
        </span>
      </ReactTooltip>
    </div>
  );
};

export default InstallStatusCell;
