import React, { ReactNode } from "react";

import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";
import { dateAgo } from "utilities/date_format";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";

const baseClass = "install-status-cell";

type IStatusValue = SoftwareInstallStatus | "avaiableForInstall";
interface TootipArgs {
  softwareName?: string | null;
  lastInstalledAt?: string;
  isAppStoreApp?: boolean;
}

export type IStatusDisplayConfig = {
  iconName:
    | "success"
    | "pending-outline"
    | "error"
    | "success-outline"
    | "install"
    | "install-self-service";
  displayText: string;
  tooltip: (args: TootipArgs) => ReactNode;
};

export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  IStatusValue | "selfService",
  IStatusDisplayConfig
> = {
  verified: {
    iconName: "success",
    displayText: "Verified",
    tooltip: ({ lastInstalledAt: lastInstall }) => (
      <>
        Fleet installed software on this host ({dateAgo(lastInstall as string)}
        ). Currently, if the software is uninstalled, the &quot;Installed&quot;
        status won&apos;t be updated.
      </>
    ),
  },
  verifying: {
    iconName: "success-outline",
    displayText: "Verifying",
    tooltip: () => (
      // TODO
      <>TODO InstallStatusCell.tsx</>
    ),
  },
  pending: {
    iconName: "pending-outline",
    displayText: "Pending",
    tooltip: () => "Fleet will install software when the host comes online.",
  },
  blocked: {
    iconName: "success", // TODO
    displayText: "Blocking",
    tooltip: () => (
      // TODO
      <>TODO InstallStatusCell.tsx</>
    ),
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
    tooltip: ({ softwareName, isAppStoreApp }) =>
      isAppStoreApp ? (
        <>
          App Store app can be installed on the host. Select{" "}
          <b>Actions {">"} Install</b> to install.
        </>
      ) : (
        <>
          {softwareName ? <b>{softwareName}</b> : "Software"} can be installed
          on the host. Select <b>Actions {">"} Install</b> to install.
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

type IInstallStatusCellProps = IHostSoftware;

const InstallStatusCell = ({
  status,
  software_package,
  app_store_app,
}: IInstallStatusCellProps) => {
  // FIXME: Improve the way we handle polymophism of software_package and app_store_app
  const lastInstalledAt =
    software_package?.last_install?.installed_at ||
    app_store_app?.last_install?.installed_at ||
    "";
  const hasPackage = !!software_package;
  const hasAppStoreApp = !!app_store_app;

  let displayStatus: keyof typeof INSTALL_STATUS_DISPLAY_OPTIONS;

  if (status !== null) {
    displayStatus = status;
  } else if (software_package?.self_service) {
    // currently only software packages can be self-service
    displayStatus = "selfService";
  } else if (hasPackage || hasAppStoreApp) {
    displayStatus = "avaiableForInstall";
  } else {
    return <TextCell value="---" grey italic />;
  }

  const displayConfig = INSTALL_STATUS_DISPLAY_OPTIONS[displayStatus];
  const tooltipId = uniqueId();
  const softwareName = software_package?.name;

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
            isAppStoreApp: hasAppStoreApp,
          })}
        </span>
      </ReactTooltip>
    </div>
  );
};

export default InstallStatusCell;
