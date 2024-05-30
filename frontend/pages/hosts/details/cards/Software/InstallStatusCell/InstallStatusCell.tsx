import React, { ReactNode } from "react";

import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import { SoftwareInstallStatus } from "interfaces/software";
import { dateAgo } from "utilities/date_format";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";

const baseClass = "install-status-cell";

type IStatusValue = SoftwareInstallStatus | "avaiableForInstall";

type IStatusDisplayConfig = {
  iconName: "success" | "pending-outline" | "error" | "install";
  displayText: string;
  tooltip: (softwareName?: string | null, lastInstall?: string) => ReactNode;
};

const CELL_DISPLAY_OPTIONS: Record<IStatusValue, IStatusDisplayConfig> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: (_, lastInstall) => (
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
    tooltip: (_, lastInstall) => (
      <>
        Fleet failed to install software ({dateAgo(lastInstall as string)} ago).
        Select <b>Actions &gt; Software details</b> to see more.
      </>
    ),
  },
  avaiableForInstall: {
    iconName: "install",
    displayText: "Available for install",
    tooltip: (softwareName) => (
      <>
        <b>{softwareName}</b> can be installed on the host. Select{" "}
        <b>Actions &gt; Install</b> to install.
      </>
    ),
  },
};

interface IInstallStatusCellProps {
  status: SoftwareInstallStatus | null;
  packageToInstall?: string | null;
  installedAt?: string;
}

const InstallStatusCell = ({
  status,
  packageToInstall,
  installedAt,
}: IInstallStatusCellProps) => {
  let displayStatus: IStatusValue;

  if (status !== null) {
    displayStatus = status;
  } else if (packageToInstall) {
    displayStatus = "avaiableForInstall";
  } else {
    return <TextCell value="---" greyed />;
  }

  const displayConfig = CELL_DISPLAY_OPTIONS[displayStatus];
  const tooltipId = uniqueId();

  return (
    <div className={`${baseClass}__status-content`}>
      <div
        className={`${baseClass}__status-with-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        <Icon name={displayConfig.iconName} />
      </div>
      <ReactTooltip
        className={`${baseClass}__status-tooltip`}
        effect="solid"
        backgroundColor="#3e4771"
        id={tooltipId}
        data-html
      >
        <span className={`${baseClass}__status-tooltip-text`}>
          {displayConfig.tooltip(packageToInstall, installedAt)}
        </span>
      </ReactTooltip>
      <span>{displayConfig.displayText}</span>
    </div>
  );
};

export default InstallStatusCell;
