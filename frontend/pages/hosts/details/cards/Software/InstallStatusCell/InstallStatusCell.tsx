import React, { ReactNode } from "react";

import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";

const baseClass = "install-status-cell";

type IStatusValue = SoftwareInstallStatus | "avaiableForInstall";
interface TootipArgs {
  softwareName?: string | null;
  // this field is used in My device > Self-service
  lastInstalledAt?: string;
  isAppStoreApp?: boolean;
}

export type IStatusDisplayConfig = {
  iconName:
    | "success"
    | "pending-outline"
    | "error"
    | "install"
    | "install-self-service";
  displayText: string;
  tooltip: (args: TootipArgs) => ReactNode;
};

export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  Exclude<IStatusValue, "uninstalled"> | "selfService",
  IStatusDisplayConfig
> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ isAppStoreApp }) =>
      isAppStoreApp
        ? "The host acknowledged the MDM command to install App Store app."
        : "Software is installed (install script finished with exit code 0).",
  },
  pending_install: {
    iconName: "pending-outline",
    displayText: "Installing (pending)",
    tooltip: () =>
      "Fleet is installing or will install when the host comes online.",
  },
  pending_uninstall: {
    iconName: "pending-outline",
    displayText: "Uninstalling (pending)",
    tooltip: () => (
      <>
        Fleet is uninstalling or will uninstall
        <br />
        software when the host comes online.
      </>
    ),
  },
  failed_install: {
    iconName: "error",
    displayText: "Install (failed)",
    tooltip: () => (
      <>
        The host failed to install software.
        <br />
        Select <b>Actions &gt; Show details</b> view errors.
      </>
    ),
  },
  failed_uninstall: {
    iconName: "error",
    displayText: "Uninstall (failed)",
    tooltip: () => (
      <>
        The host failed to uninstall software.
        <br />
        Select <b>Details &gt; Activity</b> to view errors.
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
            isAppStoreApp: hasAppStoreApp,
          })}
        </span>
      </ReactTooltip>
    </div>
  );
};

export default InstallStatusCell;
