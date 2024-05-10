import React, { ReactNode } from "react";

import { ISoftwareInstallStatus } from "interfaces/software";
import { dateAgo } from "utilities/date_format";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import TextCell from "components/TableContainer/DataTable/TextCell";

const baseClass = "install-status-cell";

type IStatusValue = ISoftwareInstallStatus | "avaiableForInstall";

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
  status: ISoftwareInstallStatus | null;
  packageToInstall?: string | null;
  installedAt?: string;
}

const InstallStatusCell = ({
  status,
  packageToInstall,
  installedAt,
}: IInstallStatusCellProps) => {
  let displayStatus: IStatusValue;

  if (packageToInstall) {
    displayStatus = "avaiableForInstall";
  } else if (status !== null) {
    displayStatus = status;
  } else {
    return <TextCell value="---" greyed />;
  }

  const displayConfig = CELL_DISPLAY_OPTIONS[displayStatus];

  return (
    <TooltipWrapper
      tipContent={displayConfig.tooltip(packageToInstall, installedAt)}
      underline={false}
      className={baseClass}
      tooltipClass={`${baseClass}__status-tooltip`}
      position="top"
    >
      <div className={`${baseClass}__status-content`}>
        <Icon name={displayConfig.iconName} />
        <span>{displayConfig.displayText}</span>
      </div>
    </TooltipWrapper>
  );
};

export default InstallStatusCell;
