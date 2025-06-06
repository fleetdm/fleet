import React, {
  useRef,
  useEffect,
  useState,
  useCallback,
  useContext,
} from "react";
import { CellProps, Column } from "react-table";

import {
  IAppLastInstall,
  IHostSoftware,
  ISoftwareLastInstall,
  ISoftwareLastUninstall,
  SoftwareInstallStatus,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import deviceApi from "services/entities/device_user";
import { NotificationContext } from "context/notification";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { dateAgo } from "utilities/date_format";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import Spinner from "components/Spinner";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { IStatusDisplayConfig } from "../InstallStatusCell/InstallStatusCell";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IStatusCellProps = CellProps<IHostSoftware, IHostSoftware["status"]>;
type IActionCellProps = CellProps<IHostSoftware, IHostSoftware["status"]>;

const baseClass = "self-service-table";

const STATUS_CONFIG: Record<SoftwareInstallStatus, IStatusDisplayConfig> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ lastInstalledAt = null }) => {
      return `Software was installed${
        lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""
      }.`;
    },
  },
  pending_install: {
    iconName: "pending-outline",
    displayText: "Installing...",
    tooltip: () => "Fleet is installing software.",
  },
  failed_install: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ lastInstalledAt = null }) => (
      <>
        Software failed to install
        {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
        <b>Retry</b> to install again, or contact your IT department.
      </>
    ),
  },
  uninstalled: {
    iconName: "success",
    displayText: "Uninstalled",
    tooltip: ({ lastInstalledAt = null }) => {
      return `Software uninstalled${
        lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""
      }.`;
    },
  },
  pending_uninstall: {
    iconName: "pending-outline",
    displayText: "Uninstalling...",
    tooltip: () => "Fleet is uninstalling software.",
  },
  failed_uninstall: {
    iconName: "error",
    displayText: "Failed (uninstall)",
    tooltip: ({ lastInstalledAt = null }) => (
      <>
        Software failed to uninstall
        {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
        <b>Retry</b> to uninstall again, or contact your IT department.
      </>
    ),
  },
};

interface CommandUuid {
  command_uuid: string;
}

interface InstallUuid {
  install_uuid: string;
}

export type InstallOrCommandUuid = CommandUuid | InstallUuid;

type IInstallerStatusProps = Pick<IHostSoftware, "status"> & {
  last_install: ISoftwareLastInstall | IAppLastInstall | null;
  last_uninstall: ISoftwareLastUninstall | null;
  onShowInstallerDetails: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (scriptExecutionId?: string) => void;
};

const InstallerStatus = ({
  status,
  last_install,
  last_uninstall,
  onShowInstallerDetails,
  onShowUninstallDetails,
}: IInstallerStatusProps) => {
  // Ensures we display info for current status even when there's both last_install and last_uninstall data
  const displayConfig = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG];

  if (!displayConfig) {
    // Empty cell value if an install/uninstall has never ran
    return <>{DEFAULT_EMPTY_CELL_VALUE}</>;
  }

  const renderDisplayStatus = () => {
    if (last_install && displayConfig.displayText === "Failed") {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-icon"
          onClick={() => {
            if ("command_uuid" in last_install) {
              onShowInstallerDetails({
                command_uuid: last_install.command_uuid,
              });
            } else if ("install_uuid" in last_install) {
              onShowInstallerDetails({
                install_uuid: last_install.install_uuid,
              });
            } else {
              onShowInstallerDetails(undefined);
            }
          }}
        >
          {displayConfig.displayText}
        </Button>
      );
    }

    if (last_uninstall && displayConfig.displayText === "Failed (uninstall)") {
      return (
        <Button
          className={`${baseClass}__item-status-button`}
          variant="text-icon"
          onClick={() => {
            if ("script_execution_id" in last_uninstall) {
              onShowUninstallDetails(
                (last_uninstall as {
                  script_execution_id: string;
                }).script_execution_id
              );
            } else {
              onShowUninstallDetails(undefined);
            }
          }}
        >
          {displayConfig.displayText}
        </Button>
      );
    }

    return displayConfig.displayText;
  };

  return (
    <TooltipWrapper
      tipContent={displayConfig.tooltip({
        lastInstalledAt: last_install?.installed_at,
      })}
      showArrow
      underline={false}
      position="top"
    >
      <div className={`${baseClass}__status-content`}>
        {displayConfig.iconName === "pending-outline" ? (
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

interface IInstallerStatusActionsProps {
  deviceToken: string;
  software: IHostSoftware;
  onInstallOrUninstall: () => void;
  onClickUninstallAction: (software: IHostSoftware) => void;
}

export interface DisplayActionItems {
  install: {
    text: string;
    icon: IconNames;
  };
  uninstall: {
    text: string;
    icon: IconNames;
  };
}

export const getInstallButtonText = (status: SoftwareInstallStatus | null) => {
  switch (status) {
    case "failed_install":
      return "Retry";
    case "installed":
    case "uninstalled":
    case "pending_uninstall":
      return "Reinstall";
    default:
      // including null
      return "Install";
  }
};

export const getInstallButtonIcon = (status: SoftwareInstallStatus | null) => {
  switch (status) {
    case "failed_install":
    case "installed":
    case "uninstalled":
    case "pending_uninstall":
    case "failed_uninstall":
      return "refresh";
    default:
      // including null
      return "install";
  }
};

export const getUninstallButtonText = (
  status: SoftwareInstallStatus | null
) => {
  switch (status) {
    case "failed_uninstall":
      return "Retry uninstall";
    default:
      // including null, "installed", "pending_install", "pending_uninstalled", "failed_install"
      return "Uninstall";
  }
};

export const getUninstallButtonIcon = (
  status: SoftwareInstallStatus | null
) => {
  switch (status) {
    case "failed_uninstall":
      return "refresh";
    default:
      // including null, "installed", "pending_install", "pending_uninstalled", "failed_install"
      return "trash";
  }
};

const InstallerStatusAction = ({
  deviceToken,
  software: { id, status, software_package, app_store_app },
  onInstallOrUninstall,
  onClickUninstallAction,
}: IInstallerStatusActionsProps) => {
  const { renderFlash } = useContext(NotificationContext);

  // displayActionItems is used to track the display text and icons of the install and uninstall button
  const [
    displayActionItems,
    setDisplayActionItems,
  ] = useState<DisplayActionItems>({
    install: {
      text: getInstallButtonText(status),
      icon: getInstallButtonIcon(status),
    },
    uninstall: {
      text: getUninstallButtonText(status),
      icon: getUninstallButtonIcon(status),
    },
  });

  useEffect(() => {
    // We update the text/icon only when we see a change to a non-pending status
    // Pending statuses keep the original text shown (e.g. "Retry" text on failed
    // install shouldn't change to "Install" text because it was clicked and went
    // pending. Once the status is no longer pending, like 'installed' the
    // text will update to "Reinstall")
    if (status !== "pending_install" && status !== "pending_uninstall") {
      setDisplayActionItems({
        install: {
          text: getInstallButtonText(status),
          icon: getInstallButtonIcon(status),
        },
        uninstall: {
          text: getUninstallButtonText(status),
          icon: getUninstallButtonIcon(status),
        },
      });
    }
  }, [status]);

  const isAppStoreApp = !!app_store_app;
  const canUninstallSoftware = !isAppStoreApp && !!software_package;

  const isMountedRef = useRef(false);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const onClickInstallAction = useCallback(async () => {
    try {
      await deviceApi.installSelfServiceSoftware(deviceToken, id);
      if (isMountedRef.current) {
        onInstallOrUninstall();
      }
    } catch (error) {
      // We only show toast message if API returns an error
      renderFlash("error", "Couldn't install. Please try again.");
    }
  }, [deviceToken, id, onInstallOrUninstall, renderFlash]);

  return (
    <div className={`${baseClass}__item-actions`}>
      <div className={`${baseClass}__item-action`}>
        <Button
          variant="text-icon"
          type="button"
          className={`${baseClass}__item-action-button`}
          onClick={onClickInstallAction}
          disabled={
            status === "pending_install" || status === "pending_uninstall"
          }
        >
          <Icon
            name={displayActionItems.install.icon}
            color="core-fleet-blue"
            size="small"
          />
          <span data-testid={`${baseClass}__install-button--test`}>
            {displayActionItems.install.text}
          </span>
        </Button>
      </div>
      <div className={`${baseClass}__item-action`}>
        {canUninstallSoftware && (
          <Button
            variant="text-icon"
            type="button"
            className={`${baseClass}__item-action-button`}
            onClick={onClickUninstallAction}
            disabled={
              status === "pending_install" || status === "pending_uninstall"
            }
          >
            <Icon
              name={displayActionItems.uninstall.icon}
              color="core-fleet-blue"
              size="small"
            />
            <span data-testid={`${baseClass}__uninstall-button--test`}>
              {displayActionItems.uninstall.text}
            </span>
          </Button>
        )}
      </div>
    </div>
  );
};

export const generateSoftwareTableData = (
  software: IHostSoftware[]
): IHostSoftware[] => {
  return software;
};

interface ISelfServiceTableHeaders {
  deviceToken: string;
  onInstall: () => void;
  onShowInstallerDetails: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (scriptExecutionId?: string) => void;
  onClickUninstallAction: (software: IHostSoftware) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  deviceToken,
  onInstall,
  onShowInstallerDetails,
  onShowUninstallDetails,
  onClickUninstallAction,
}: ISelfServiceTableHeaders): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      disableGlobalFilter: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, source, app_store_app } = cellProps.row.original;
        return (
          <SoftwareNameCell
            name={name}
            source={source}
            iconUrl={app_store_app?.icon_url}
            myDevicePage
            isSelfService
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Install status"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      disableGlobalFilter: true,
      accessor: "status",
      Cell: (cellProps: IStatusCellProps) => (
        <InstallerStatus
          status={cellProps.row.original.status}
          last_install={
            cellProps.row.original.software_package?.last_install ||
            cellProps.row.original.app_store_app?.last_install ||
            null
          }
          last_uninstall={
            cellProps.row.original.software_package?.last_uninstall || null
          }
          onShowInstallerDetails={onShowInstallerDetails}
          onShowUninstallDetails={onShowUninstallDetails}
        />
      ),
    },
    {
      Header: "Actions",
      accessor: (originalRow) => originalRow.status,
      disableSortBy: true,
      Cell: (cellProps: IActionCellProps) => {
        return (
          <InstallerStatusAction
            deviceToken={deviceToken}
            software={cellProps.row.original}
            onInstallOrUninstall={onInstall}
            onClickUninstallAction={() =>
              onClickUninstallAction(cellProps.row.original)
            }
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders, generateSoftwareTableData };
