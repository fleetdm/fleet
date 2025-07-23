import React, { ReactNode } from "react";

import { dateAgo } from "utilities/date_format";
import {
  IHostSoftware,
  ISoftwareAppStoreAppStatus,
  SoftwareInstallStatus,
} from "interfaces/software";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

const baseClass = "install-status-cell";

interface CommandUuid {
  command_uuid: string;
  software_title?: string;
  status?: SoftwareInstallStatus;
}

interface InstallUuid {
  install_uuid: string;
}

export type InstallOrCommandUuid = CommandUuid | InstallUuid;

type IStatusValue = SoftwareInstallStatus;
interface DisplayTextArgs {
  isSelfService?: boolean;
  isHostOnline?: boolean;
}
interface TooltipArgs {
  isSelfService?: boolean;
  softwareName?: string | null;
  lastInstalledAt?: string;
  isAppStoreApp?: boolean;
  isHostOnline?: boolean;
}

export type IStatusDisplayConfig = {
  iconName?: "success" | "pending-outline" | "error" | "install";
  displayText: string | ((args: DisplayTextArgs) => React.ReactNode);
  tooltip: (args: TooltipArgs) => ReactNode;
};

// Similar to SelfServiceTableConfig STATUS_CONFIG
export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  Exclude<IStatusValue, "uninstalled">,
  IStatusDisplayConfig
> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ isSelfService, isAppStoreApp, lastInstalledAt }) =>
      isAppStoreApp ? (
        <>
          The host acknowledged the MDM
          <br />
          command to install the app.
        </>
      ) : (
        <>
          Software was installed
          {!isSelfService && " (install script finished with exit code 0)"}
          {lastInstalledAt && ` ${dateAgo(lastInstalledAt)}`}.
        </>
      ),
  },
  pending_install: {
    iconName: "pending-outline",
    displayText: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? "Installing..." : "Install (pending)",
    tooltip: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? (
        "Fleet is installing software."
      ) : (
        <>
          Fleet will install software
          <br /> when the host comes online.
        </>
      ),
  },
  pending_uninstall: {
    iconName: "pending-outline",
    displayText: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? "Uninstalling..." : "Uninstall (pending)",
    tooltip: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? (
        "Fleet is uninstalling software."
      ) : (
        <>
          Fleet will uninstall software
          <br />
          when the host comes online.
        </>
      ),
  },
  failed_install: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ lastInstalledAt = null, isSelfService }) => (
      <>
        Software failed to install
        {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}.{" "}
        {isSelfService ? (
          <>
            Select <b>Retry</b> to install again, or contact your IT department.
          </>
        ) : (
          <>
            Select <b>Details &gt; Activity</b> to view errors.
          </>
        )}
      </>
    ),
  },
  failed_uninstall: {
    iconName: "error",
    displayText: "Failed (uninstall)",
    tooltip: ({ lastInstalledAt = null, isSelfService }) => (
      <>
        Software failed to uninstall
        {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
        <b>Retry</b> to uninstall again
        {isSelfService && ", or contact your IT department"}.
      </>
    ),
  },
};

type IInstallStatusCellProps = {
  software: IHostSoftware;
  onShowSoftwareDetails?: (software: IHostSoftware) => void;
  onShowInstallDetails?: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (details?: ISoftwareUninstallDetails) => void;
  isSelfService?: boolean;
  isHostOnline?: boolean;
  hostName?: string;
};

const getLastInstall = (software: IHostSoftware) =>
  software.software_package?.last_install ||
  software.app_store_app?.last_install ||
  null;

const getLastUninstall = (software: IHostSoftware) =>
  software.software_package?.last_uninstall || null;

const getSoftwareName = (software: IHostSoftware) =>
  software.software_package?.name;

const resolveDisplayText = (
  displayText: IStatusDisplayConfig["displayText"],
  isSelfService: boolean,
  isHostOnline: boolean
) =>
  typeof displayText === "function"
    ? displayText({ isSelfService, isHostOnline })
    : displayText;

const getEmptyCellTooltip = (hasAppStoreApp: boolean, softwareName?: string) =>
  hasAppStoreApp ? (
    <>
      App Store app can be installed on the host. <br />
      Select <b>Actions &gt; Install</b> to install.
    </>
  ) : (
    <>
      {softwareName ? <b>{softwareName}</b> : "Software"} can be installed on
      the host.
      <br /> Select <b>Actions &gt; Install</b> to install.
    </>
  );

const InstallStatusCell = ({
  software,
  onShowSoftwareDetails,
  onShowInstallDetails,
  onShowUninstallDetails,
  isSelfService = false,
  isHostOnline = false,
  hostName,
}: IInstallStatusCellProps) => {
  const hasAppStoreApp = !!software.app_store_app;
  const lastInstall = getLastInstall(software);
  const lastUninstall = getLastUninstall(software);
  const softwareName = getSoftwareName(software);

  const displayStatus = software.status as
    | keyof typeof INSTALL_STATUS_DISPLAY_OPTIONS
    | null;

  // Status is null
  if (displayStatus === null) {
    return (
      <TextCell
        grey
        italic
        emptyCellTooltipText={getEmptyCellTooltip(hasAppStoreApp, softwareName)}
      />
    );
  }

  const displayConfig = INSTALL_STATUS_DISPLAY_OPTIONS[displayStatus];

  const onClickInstallStatus = () => {
    if (onShowInstallDetails && lastInstall) {
      if ("command_uuid" in lastInstall) {
        onShowInstallDetails({
          command_uuid: lastInstall.command_uuid,
          software_title: software.name,
          status: software.status || undefined,
        });
      } else if ("install_uuid" in lastInstall) {
        onShowInstallDetails({ install_uuid: lastInstall.install_uuid });
      } else {
        onShowInstallDetails(undefined);
      }
    } else if (onShowSoftwareDetails) {
      onShowSoftwareDetails(software);
    }
  };

  const onClickUninstallStatus = () => {
    if (onShowUninstallDetails && lastUninstall) {
      if ("script_execution_id" in lastUninstall) {
        onShowUninstallDetails({
          ...lastUninstall,
          status: software.status || undefined,
          software_title: software.name,
          host_display_name: hostName,
        });
      } else {
        onShowUninstallDetails(undefined);
      }
    } else if (onShowSoftwareDetails) {
      onShowSoftwareDetails(software);
    }
  };

  const renderDisplayStatus = () => {
    const resolvedDisplayText = resolveDisplayText(
      displayConfig.displayText,
      isSelfService,
      isHostOnline
    );

    if (
      lastInstall &&
      (resolvedDisplayText === "Failed" ||
        resolvedDisplayText === "Install (pending)" ||
        resolvedDisplayText === "Installed")
    ) {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-link"
          onClick={onClickInstallStatus}
        >
          {resolvedDisplayText}
        </Button>
      );
    }

    if (
      lastUninstall &&
      (resolvedDisplayText === "Failed (uninstall)" ||
        resolvedDisplayText === "Uninstall (pending)" ||
        resolvedDisplayText === "Uninstalled")
    ) {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-link"
          onClick={onClickUninstallStatus}
        >
          {resolvedDisplayText}
        </Button>
      );
    }

    // Defaults to text without modal button if:
    // - there is neither last_install or last_uninstall information regardless of status
    // - Display text is "Installing...", "Uninstalling..." (host is online/self-service)
    return resolvedDisplayText;
  };

  return (
    <TooltipWrapper
      tipContent={displayConfig.tooltip({
        lastInstalledAt: lastInstall?.installed_at,
        softwareName,
        isAppStoreApp: hasAppStoreApp,
        isSelfService,
        isHostOnline,
      })}
      showArrow
      underline={false}
      position="top"
      className={`${baseClass}__tooltip-wrapper`}
    >
      <div className={baseClass}>
        {(isSelfService || isHostOnline) &&
        displayConfig.iconName === "pending-outline" ? (
          <Spinner size="x-small" includeContainer={false} centered={false} />
        ) : (
          displayConfig?.iconName && <Icon name={displayConfig.iconName} />
        )}
        <span data-testid={`${baseClass}__status--test`}>
          {renderDisplayStatus()}
        </span>
      </div>
    </TooltipWrapper>
  );
};

export default InstallStatusCell;
