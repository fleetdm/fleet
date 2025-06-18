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
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import Button from "components/buttons/Button";

import InstallerStatusCell, {
  IStatusDisplayConfig,
  InstallOrCommandUuid,
} from "../InstallStatusCell/InstallStatusCell";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IStatusCellProps = CellProps<IHostSoftware, IHostSoftware["status"]>;
type IActionCellProps = CellProps<IHostSoftware, IHostSoftware["status"]>;

const baseClass = "self-service-table";

// Similar to HostSoftwareLibraryTableConfig INSTALL_STATUS_DISPLAY_OPTIONS
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
  pending_uninstall: {
    iconName: "pending-outline",
    displayText: "Uninstalling...",
    tooltip: () => "Fleet is uninstalling software.",
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

interface ISSInstallerActionCellProps {
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

// TODO: Create and move into /components directory
const SSInstallerActionCell = ({
  deviceToken,
  software: { id, status, software_package, app_store_app },
  onInstallOrUninstall,
  onClickUninstallAction,
}: ISSInstallerActionCellProps) => {
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
  onShowInstallDetails: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (scriptExecutionId?: string) => void;
  onClickUninstallAction: (software: IHostSoftware) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  deviceToken,
  onInstall,
  onShowInstallDetails,
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
        <InstallerStatusCell
          software={cellProps.row.original}
          onShowSSInstallDetails={onShowInstallDetails}
          onShowSSUninstallDetails={onShowUninstallDetails}
          isSelfService
        />
      ),
    },
    {
      Header: "Actions",
      accessor: (originalRow) => originalRow.status,
      disableSortBy: true,
      Cell: (cellProps: IActionCellProps) => {
        return (
          <SSInstallerActionCell
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
