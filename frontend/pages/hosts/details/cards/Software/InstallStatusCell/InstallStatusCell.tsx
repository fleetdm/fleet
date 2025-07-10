import React, { ReactNode } from "react";

import { dateAgo } from "utilities/date_format";
import {
  IHostSoftware,
  IHostSoftwareWithUiStatus,
  IHostSoftwareUiStatus,
} from "interfaces/software";
import { Colors } from "styles/var/colors";

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
  iconName?:
    | "success"
    | "pending-outline"
    | "error"
    | "install"
    | "error-outline";
  iconColor?: Colors;
  displayText: string | ((args: DisplayTextArgs) => React.ReactNode);
  tooltip: (args: TooltipArgs) => ReactNode;
};

// Similar to SelfServiceTableConfig STATUS_CONFIG
export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  Exclude<IHostSoftwareUiStatus, "uninstalled">, // Uninstalled is handled separately with empty cell
  IStatusDisplayConfig
> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ isSelfService, isAppStoreApp, lastInstalledAt }) => {
      if (!lastInstalledAt) {
        return undefined;
      }

      return isAppStoreApp ? (
        <>
          The host acknowledged the MDM
          <br />
          command to install the app.
        </>
      ) : (
        <>
          Software was installed{" "}
          {!isSelfService && "(install script finished with exit code 0) "}
          {dateAgo(lastInstalledAt)}.
        </>
      );
    },
  },
  installing: {
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
  uninstalling: {
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
  pending_update: {
    iconName: "pending-outline",
    displayText: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? "Updating..." : "Update (pending)",
    tooltip: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? (
        "Fleet is updating software."
      ) : (
        <>
          Fleet will update software
          <br /> when the host comes online.
        </>
      ),
  },
  updating: {
    iconName: "pending-outline",
    displayText: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? "Updating..." : "Update (pending)",
    tooltip: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? (
        "Fleet is updating software."
      ) : (
        <>
          Fleet will update software
          <br /> when the host comes online.
        </>
      ),
  },
  update_available: {
    iconName: "error-outline",
    iconColor: "ui-fleet-black-50",
    displayText: "Update available",
    tooltip: ({ isSelfService, isHostOnline }) =>
      isSelfService || isHostOnline ? (
        "Fleet can update software."
      ) : (
        <>
          Fleet can update software
          <br /> when the host comes online.
        </>
      ),
  },
};

type IInstallStatusCellProps = {
  software: IHostSoftwareWithUiStatus;
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
  const displayStatus = software.ui_status;

  if (displayStatus === "uninstalled") {
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

  const onClickUpdateAvailableStatus = () => {
    // onShowUpdateDetails(software);
    // TODO: Finish implementation to show new update details modal
  };

  const renderDisplayStatus = () => {
    const resolvedDisplayText = resolveDisplayText(
      displayConfig.displayText,
      isSelfService,
      isHostOnline
    );

    // Status groups and their click handlers
    const displayStatusConfig = [
      {
        condition: lastInstall,
        statuses: ["Failed", "Install (pending)", "Installed"],
        onClick: onClickInstallStatus,
      },
      {
        condition: lastUninstall,
        statuses: ["Failed (uninstall)", "Uninstall (pending)", "Uninstalled"],
        onClick: onClickUninstallStatus,
      },
      {
        condition: true,
        statuses: ["Update available"],
        onClick: onClickUpdateAvailableStatus,
      },
    ];

    // Find a matching config for the current display text
    const match = displayStatusConfig.find(
      ({ condition, statuses }) =>
        condition && statuses.includes(resolvedDisplayText as string)
    );

    if (match) {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-link"
          onClick={match.onClick}
        >
          {resolvedDisplayText}
        </Button>
      );
    }

    // Default: plain text
    return resolvedDisplayText;
  };

  const tooltipContent = displayConfig.tooltip({
    lastInstalledAt: lastInstall?.installed_at,
    softwareName,
    isAppStoreApp: hasAppStoreApp,
    isSelfService,
    isHostOnline,
  });

  return (
    <TooltipWrapper
      tipContent={tooltipContent}
      showArrow
      underline={false}
      position="top"
      className={`${baseClass}__tooltip-wrapper`}
      disableTooltip={!tooltipContent}
    >
      <div className={baseClass}>
        {(isSelfService || isHostOnline) &&
        displayConfig.iconName === "pending-outline" ? (
          <Spinner size="x-small" includeContainer={false} centered={false} />
        ) : (
          displayConfig?.iconName && (
            <Icon
              name={displayConfig.iconName}
              color={displayConfig.iconColor}
            />
          )
        )}
        <span data-testid={`${baseClass}__status--test`}>
          {renderDisplayStatus()}
        </span>
      </div>
    </TooltipWrapper>
  );
};

export default InstallStatusCell;
