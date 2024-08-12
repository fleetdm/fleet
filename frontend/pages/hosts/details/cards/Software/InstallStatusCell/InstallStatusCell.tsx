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
    | "success-outline"
    | "pending-outline"
    | "disable"
    | "error"
    | "install"
    | "install-self-service";
  displayText: string;
  tooltip: (args: TootipArgs) => ReactNode;
};

/** Status used for both Install status cell and self-service item */
export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  IStatusValue | "selfService",
  IStatusDisplayConfig
> = {
  verified: {
    iconName: "success",
    displayText: "Verified",
    tooltip: ({ lastInstalledAt: lastInstall }) => (
      <>
        Software is installed ({dateAgo(lastInstall as string)}
        ). Fleet verified.
      </>
    ),
  },
  verifying: {
    iconName: "success-outline",
    displayText: "Verifying",
    tooltip: () =>
      "Software is installed (install script finished with exit code 0). Fleet is verifying.",
  },
  pending: {
    iconName: "pending-outline",
    displayText: "Pending",
    tooltip: (isAutomaticInstall) => {
      return isAutomaticInstall
        ? "Fleet is checking if the software is installed and if not, Fleet is installing or will install when the host comes online."
        : "Fleet is installing or will install when the host comes online.";
    },
  },
  blocked: {
    iconName: "disable",
    displayText: "Blocking",
    tooltip: () =>
      "Pre-install condition wasn't met. The query didn't return results.",
  },
  failed: {
    iconName: "error",
    displayText: "Failed",
    tooltip: () => (
      <>
        The host failed to install software. To view errors, select{" "}
        <b>Actions &gt; Show details</b>.
      </>
    ),
  },
  /** Used only for Install status cell */
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
  /** Used only for Install status cell */
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
