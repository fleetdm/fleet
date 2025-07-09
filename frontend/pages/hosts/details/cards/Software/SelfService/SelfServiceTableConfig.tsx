import React, {
  useRef,
  useEffect,
  useState,
  useCallback,
  useContext,
} from "react";
import { CellProps, Column } from "react-table";

import {
  IHostSoftwareUiStatus,
  IHostSoftwareWithUiStatus,
  SoftwareInstallStatus,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import deviceApi from "services/entities/device_user";
import { NotificationContext } from "context/notification";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { dateAgo } from "utilities/date_format";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import Button from "components/buttons/Button";

import InstallerStatusCell, {
  InstallOrCommandUuid,
} from "../InstallStatusCell/InstallStatusCell";

type ISoftwareTableConfig = Column<IHostSoftwareWithUiStatus>;
type ITableHeaderProps = IHeaderProps<IHostSoftwareWithUiStatus>;
type ITableStringCellProps = IStringCellProps<IHostSoftwareWithUiStatus>;
type IStatusCellProps = CellProps<
  IHostSoftwareWithUiStatus,
  IHostSoftwareWithUiStatus["ui_status"]
>;
type IActionCellProps = CellProps<
  IHostSoftwareWithUiStatus,
  IHostSoftwareWithUiStatus["status"]
>;

const baseClass = "self-service-table";

interface ISSInstallerActionCellProps {
  deviceToken: string;
  software: IHostSoftwareWithUiStatus;
  onInstallOrUninstall: () => void;
  onClickUninstallAction: (software: IHostSoftwareWithUiStatus) => void;
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

export const getInstallButtonText = (status: IHostSoftwareUiStatus | null) => {
  switch (status) {
    case "failed_install":
      return "Retry";
    case "installed":
    case "pending_uninstall":
    case "failed_uninstall":
      return "Reinstall";
    case "pending_update":
    case "update_available":
      return "Update";
    default:
      // including null
      return "Install";
  }
};

export const getInstallButtonIcon = (status: IHostSoftwareUiStatus | null) => {
  switch (status) {
    case "failed_install":
    case "installed":
    case "pending_uninstall":
    case "failed_uninstall":
    case "pending_update":
    case "update_available":
      return "refresh";
    default:
      // including null
      return "install";
  }
};

export const getUninstallButtonText = (
  status: IHostSoftwareUiStatus | null
) => {
  switch (status) {
    case "failed_uninstall":
      return "Retry uninstall";
    default:
      // including null, "installed", "pending_install", "pending_uninstall", "failed_install"
      return "Uninstall";
  }
};

export const getUninstallButtonIcon = (
  status: IHostSoftwareUiStatus | null
) => {
  switch (status) {
    case "failed_uninstall":
      return "refresh";
    default:
      // including null, "installed", "pending_install", "pending_uninstall", "failed_install"
      return "trash";
  }
};

// TODO: Create and move into /components directory
const SSInstallerActionCell = ({
  deviceToken,
  software: { id, status, software_package, app_store_app, installed_versions },
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
  const canUninstallSoftware =
    !isAppStoreApp &&
    !!software_package &&
    installed_versions &&
    installed_versions.length > 0;

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
  software: IHostSoftwareWithUiStatus[]
): IHostSoftwareWithUiStatus[] => {
  return software;
};

interface ISelfServiceTableHeaders {
  deviceToken: string;
  onInstall: () => void;
  onShowInstallDetails: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (details?: ISoftwareUninstallDetails) => void;
  onClickUninstallAction: (software: IHostSoftwareWithUiStatus) => void;
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
            pageContext="deviceUser"
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
      // sortType: (row1, row2) => {
      //   const status1 = row1.original.status;
      //   const status2 = row2.original.status;

      //   // Sort by the order of statuses defined in STATUS_CONFIG
      //   const statusOrder = Object.keys(STATUS_CONFIG);
      //   return (
      //     statusOrder.indexOf(status1) - statusOrder.indexOf(status2)
      //   );
      // }
      disableSortBy: false,
      disableGlobalFilter: true,
      accessor: "ui_status",
      Cell: (cellProps: IStatusCellProps) => (
        <InstallerStatusCell
          software={cellProps.row.original}
          onShowInstallDetails={onShowInstallDetails}
          onShowUninstallDetails={onShowUninstallDetails}
          isSelfService
        />
      ),
    },
    {
      Header: "Actions",
      accessor: "status",
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
