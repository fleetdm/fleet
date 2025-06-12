import React, { ReactNode } from "react";

import { dateAgo } from "utilities/date_format";
import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";

const baseClass = "install-status-cell";

interface CommandUuid {
  command_uuid: string;
}

interface InstallUuid {
  install_uuid: string;
}

export type InstallOrCommandUuid = CommandUuid | InstallUuid;

type IStatusValue = SoftwareInstallStatus;
interface TootipArgs {
  isSelfService?: boolean;
  softwareName?: string | null;
  lastInstalledAt?: string;
  isAppStoreApp?: boolean;
}

export type IStatusDisplayConfig = {
  iconName?: "success" | "pending-outline" | "error" | "install";
  displayText: string | JSX.Element;
  tooltip: (args: TootipArgs) => ReactNode;
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
          Software was installed{" "}
          {!isSelfService && `(install script finished with exit code 0)`}
          {lastInstalledAt ? ` ${dateAgo(lastInstalledAt)}.` : "."}
        </>
      ),
  },
  pending_install: {
    iconName: "pending-outline",
    displayText: "Installing...",
    tooltip: ({ isSelfService }) =>
      isSelfService ? (
        "Fleet is installing software."
      ) : (
        <>
          Fleet is installing or will install
          <br /> when the host comes online.
        </>
      ),
  },
  pending_uninstall: {
    iconName: "pending-outline",
    displayText: "Uninstalling...",
    tooltip: ({ isSelfService }) =>
      isSelfService ? (
        "Fleet is installing software."
      ) : (
        <>
          Fleet is uninstalling or will uninstall
          <br />
          software when the host comes online.
        </>
      ),
  },
  failed_install: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ lastInstalledAt = null, isSelfService }) => (
      <>
        Software failed to install
        {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
        <b>Retry</b> to install again
        {isSelfService && ", or contact your IT department"}.
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
        <b>Retry</b> to uninstall again{" "}
        {isSelfService && ", or contact your IT department"}.
      </>
    ),
  },
};

type IInstallStatusCellProps = {
  software: IHostSoftware;
  onShowSoftwareDetails?: (software: IHostSoftware) => void;
  onShowSSInstallDetails?: (uuid?: InstallOrCommandUuid) => void;
  onShowSSUninstallDetails?: (scriptExecutionId?: string) => void;
  isSelfService?: boolean;
};

const InstallStatusCell = ({
  software,
  onShowSoftwareDetails,
  onShowSSInstallDetails,
  onShowSSUninstallDetails,
  isSelfService = false,
}: IInstallStatusCellProps) => {
  const { app_store_app, software_package, status } = software;
  // FIXME: Improve the way we handle polymophism of software_package and app_store_app
  // const hasPackage = !!software_package;
  const hasAppStoreApp = !!app_store_app;
  const lastInstall =
    software_package?.last_install || app_store_app?.last_install || null;
  const lastUninstall = software_package?.last_uninstall || null;

  let displayStatus: keyof typeof INSTALL_STATUS_DISPLAY_OPTIONS;

  if (status !== null) {
    displayStatus = status;
  } else {
    return (
      <TextCell
        value={undefined}
        grey
        italic
        emptyCellTooltipText={
          hasAppStoreApp ? (
            <>
              App Store app can be installed on the host. <br />
              Select <b>Actions &gt; Install</b> to install.
            </>
          ) : (
            <>
              {software_package?.name ? (
                <b>{software_package.name}</b>
              ) : (
                "Software"
              )}{" "}
              can be installed on the host.
              <br /> Select <b>Actions &gt; Install</b> to install.
            </>
          )
        }
      />
    );
  }

  const displayConfig = INSTALL_STATUS_DISPLAY_OPTIONS[displayStatus];

  const renderDisplayStatus = () => {
    if (lastInstall && displayConfig.displayText === "Failed") {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-icon"
          onClick={() => {
            if (isSelfService && onShowSSInstallDetails) {
              if ("command_uuid" in lastInstall) {
                onShowSSInstallDetails({
                  command_uuid: lastInstall.command_uuid,
                });
              } else if ("install_uuid" in lastInstall) {
                onShowSSInstallDetails({
                  install_uuid: lastInstall.install_uuid,
                });
              } else {
                onShowSSInstallDetails(undefined);
              }
            } else {
              onShowSoftwareDetails && onShowSoftwareDetails(software);
            }
          }}
        >
          {displayConfig.displayText}
        </Button>
      );
    }

    if (lastUninstall && displayConfig.displayText === "Failed (uninstall)") {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-icon"
          onClick={() => {
            if (isSelfService && onShowSSUninstallDetails) {
              // If the last uninstall has a script_execution_id, we pass it to the handler
              if ("script_execution_id" in lastUninstall) {
                onShowSSUninstallDetails(
                  (lastUninstall as {
                    script_execution_id: string;
                  }).script_execution_id
                );
              } else {
                onShowSSUninstallDetails(undefined);
              }
            } else {
              onShowSoftwareDetails && onShowSoftwareDetails(software);
            }
          }}
        >
          {displayConfig.displayText}
        </Button>
      );
    }

    return displayConfig.displayText;
  };

  const softwareName = software_package?.name;

  return (
    <TooltipWrapper
      tipContent={displayConfig.tooltip({
        lastInstalledAt: lastInstall?.installed_at,
        softwareName,
        isAppStoreApp: hasAppStoreApp,
      })}
      showArrow
      underline={false}
      position="top"
    >
      <div className={`${baseClass}__status-content`}>
        {isSelfService && displayConfig.iconName === "pending-outline" ? (
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
