import React, { ReactNode } from "react";

import { dateAgo } from "utilities/date_format";
import {
  IHostSoftware,
  IHostSoftwareWithUiStatus,
  IHostSoftwareUiStatus,
  SoftwareInstallStatus,
  IVPPHostSoftware,
  SoftwareUninstallStatus,
  IAppLastInstall,
} from "interfaces/software";
import { Colors } from "styles/var/colors";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import { ISWUninstallDetailsParentState } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import {
  getLastInstall,
  getLastUninstall,
} from "../../HostSoftwareLibrary/helpers";

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

export const RECENT_SUCCESS_ACTION_MESSAGE = (
  action: "installed" | "uninstalled" | "updated"
) =>
  `Fleet successfully ${action} software and is fetching latest software inventory.`;

// Similar to SelfServiceTableConfig STATUS_CONFIG
export const INSTALL_STATUS_DISPLAY_OPTIONS: Record<
  Exclude<IHostSoftwareUiStatus, "uninstalled" | "recently_uninstalled">, // Uninstalled/recently uninstalled is handled separately with empty cell
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
  recently_updated: {
    iconName: "success",
    displayText: "Updated",
    tooltip: () => {
      return RECENT_SUCCESS_ACTION_MESSAGE("updated");
    },
  },
  recently_installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: () => {
      return RECENT_SUCCESS_ACTION_MESSAGE("installed");
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
          !lastInstalledAt && (
            <>
              Select <b>Details &gt; Activity</b> to view errors.
            </>
          )
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
  failed_install_update_available: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ isSelfService, isHostOnline, lastInstalledAt }) =>
      isSelfService || isHostOnline ? (
        <>
          Software failed to install
          {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}.{" "}
          {isSelfService ? (
            <>
              Select <b>Retry</b> to install again, or contact your IT
              department.
            </>
          ) : (
            <>
              Select <b>Details &gt; Activity</b> to view errors.
            </>
          )}
        </>
      ) : (
        <>
          Software failed to install
          {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}.{" "}
          {isSelfService ? (
            <>
              Select <b>Retry</b> to install again, or contact your IT
              department.
            </>
          ) : (
            <>
              Select <b>Details &gt; Activity</b> to view errors.
            </>
          )}
        </>
      ),
  },
  failed_uninstall_update_available: {
    iconName: "error",
    displayText: "Failed (uninstall)",
    tooltip: ({ isSelfService, isHostOnline, lastInstalledAt }) =>
      isSelfService || isHostOnline ? (
        <>
          Software failed to uninstall
          {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
          <b>Retry</b> to uninstall again
          {isSelfService && ", or contact your IT department"}.
        </>
      ) : (
        <>
          {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
          <b>Retry</b> to uninstall again
          {isSelfService && ", or contact your IT department"}.
        </>
      ),
  },
};

type IInstallStatusCellProps = {
  software: IHostSoftwareWithUiStatus;
  onShowInventoryVersions?: (software: IHostSoftware) => void;
  onShowUpdateDetails: (software: IHostSoftware) => void;
  onShowInstallDetails: (hostSoftware: IHostSoftware) => void;
  onShowVPPInstallDetails: (s: IVPPHostSoftware) => void;
  onShowUninstallDetails: (details: ISWUninstallDetailsParentState) => void;
  isSelfService?: boolean;
  isHostOnline?: boolean;
};

const getSoftwarePackageName = (software: IHostSoftware) =>
  software.software_package?.name;

const resolveDisplayText = (
  displayText: IStatusDisplayConfig["displayText"],
  isSelfService: boolean,
  isHostOnline: boolean
) =>
  typeof displayText === "function"
    ? displayText({ isSelfService, isHostOnline })
    : displayText;

const getEmptyCellTooltip = (isAppStoreApp: boolean, softwareName?: string) =>
  isAppStoreApp ? (
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
  onShowInventoryVersions,
  onShowUpdateDetails,
  onShowInstallDetails,
  onShowVPPInstallDetails,
  onShowUninstallDetails,
  isSelfService = false,
  isHostOnline = false,
}: IInstallStatusCellProps) => {
  const isAppStoreApp = !!software.app_store_app;
  const lastInstall = getLastInstall(software); // TODO (back end bug fix) - `software.app_store_app.last_install sometimes coming back `null` for VPP apps, currently falls back to displaying the `InventoryVersionsModal`
  const lastUninstall = getLastUninstall(software);
  const softwarePackageName = getSoftwarePackageName(software); // @RachelElysia I renamed this function and the variable name its return value is set to here because it is looking at the software_package.name, which has a suffix like ".pkg". software.name has the more human-readable version. Not sure how else this data is being used so I am not going to refactor anything. Please update if needed.
  const displayStatus = software.ui_status;

  if (displayStatus === "uninstalled") {
    return (
      <TextCell
        grey
        italic
        emptyCellTooltipText={getEmptyCellTooltip(
          isAppStoreApp,
          softwarePackageName
        )}
      />
    );
  }

  if (displayStatus === "recently_uninstalled") {
    return (
      <TextCell
        grey
        italic
        emptyCellTooltipText={RECENT_SUCCESS_ACTION_MESSAGE("uninstalled")}
      />
    );
  }

  const displayConfig = INSTALL_STATUS_DISPLAY_OPTIONS[displayStatus];

  // This is never called for App Store app missing 'last_install' info for
  // successful and failed installs (Old clients <4.72 bug) See shouldOnClickBeDisabled
  const onClickInstallStatus = () => {
    // VPP Install details modal will handle command_uuid missing gracefully for pending installs, etc
    if (isAppStoreApp) {
      onShowVPPInstallDetails({
        ...software,
        ...(lastInstall && {
          commandUuid: (lastInstall as IAppLastInstall).command_uuid,
        }),
      });
    } else {
      onShowInstallDetails(software);
    }
  };

  const onClickUninstallStatus = () => {
    if (lastUninstall) {
      if ("script_execution_id" in lastUninstall) {
        onShowUninstallDetails({
          softwareName: software.name || "",
          softwarePackageName,
          uninstallStatus: (software.status ||
            "pending_uninstall") as SoftwareUninstallStatus,
          scriptExecutionId: lastUninstall.script_execution_id,
          hostSoftware: software,
        });
      }
    } else if (onShowInventoryVersions) {
      onShowInventoryVersions(software);
    }
  };

  const onClickUpdateAvailableStatus = () => {
    onShowUpdateDetails(software);
  };

  const renderDisplayStatus = () => {
    const resolvedDisplayText = resolveDisplayText(
      displayConfig.displayText,
      isSelfService,
      isHostOnline
    );

    // Software "installed" by Fleet (backend) and shows as "installed" (UI)
    const isInstalledInFleetAndUI =
      software.status === "installed" && software.ui_status === "installed";

    // Is this an App Store app missing 'last_install' info? (Old clients <4.72 bug)
    const isMissingLastInstallInfo =
      isAppStoreApp && !software.app_store_app?.last_install;

    // These temporary statuses are not clickable because it will show outdated info in modal
    const recentlyTakenAction =
      software.ui_status === "recently_installed" ||
      software.ui_status === "recently_updated" ||
      software.ui_status === "recently_uninstalled";

    const shouldOnClickBeDisabled =
      (isMissingLastInstallInfo &&
        (software.status === "failed_install" || isInstalledInFleetAndUI)) ||
      recentlyTakenAction;

    // Status groups and their click handlers
    const displayStatusConfig = [
      {
        condition: true, // Allow click even if no last install to see details modal
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
        statuses: ["Update available", "Update (pending)"],
        onClick: onClickUpdateAvailableStatus,
      },
    ];

    // Find a matching config for the current display text
    const match = displayStatusConfig.find(
      ({ condition, statuses }) =>
        condition && statuses.includes(resolvedDisplayText as string)
    );

    if (match && !shouldOnClickBeDisabled) {
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
    softwareName: softwarePackageName,
    isAppStoreApp,
    isSelfService,
    isHostOnline,
  });

  return (
    <div className={baseClass}>
      <TooltipWrapper
        tipContent={tooltipContent}
        showArrow
        underline={false}
        position="top"
        className={`${baseClass}__tooltip-wrapper`}
        disableTooltip={!tooltipContent}
      >
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
      </TooltipWrapper>
    </div>
  );
};

export default InstallStatusCell;
