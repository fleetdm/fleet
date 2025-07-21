import React, {
  useRef,
  useEffect,
  useState,
  useCallback,
  useContext,
} from "react";
import { CellProps, Column, Row } from "react-table";

import { IHostSoftwareWithUiStatus } from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import deviceApi from "services/entities/device_user";
import { NotificationContext } from "context/notification";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";

import InstallerStatusCell, {
  InstallOrCommandUuid,
} from "../InstallStatusCell/InstallStatusCell";
import {
  getInstallerActionButtonConfig,
  IButtonDisplayConfig,
  installStatusSortType,
} from "../helpers";
import { HostInstallerActionButton } from "../../HostSoftwareLibrary/HostInstallerActionCell/HostInstallerActionCell";

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

interface ISSInstallerActionCellProps {
  deviceToken: string;
  software: IHostSoftwareWithUiStatus;
  onInstallOrUninstall: () => void;
  onClickUninstallAction: () => void;
}

interface ISSInstallerActionCellProps {
  deviceToken: string;
  software: IHostSoftwareWithUiStatus;
  onInstallOrUninstall: () => void;
  onClickUninstallAction: () => void;
}

const baseClass = "self-service-table";

/** Self-service installer action cell component has different disabled states
 * and tooltips than Host details > Library HostInstallerActionCell.
 * Future iterations consider combining the two */
const SSInstallerActionCell = ({
  deviceToken,
  software: {
    id,
    status,
    software_package,
    app_store_app,
    installed_versions,
    ui_status,
  },
  onInstallOrUninstall,
  onClickUninstallAction,
}: ISSInstallerActionCellProps) => {
  const { renderFlash } = useContext(NotificationContext);

  // buttonDisplayConfig is used to track the display text and icons of the install and uninstall button
  const [
    buttonDisplayConfig,
    setButtonDisplayConfig,
  ] = useState<IButtonDisplayConfig>({
    install: getInstallerActionButtonConfig("install", ui_status),
    uninstall: getInstallerActionButtonConfig("uninstall", ui_status),
  });

  useEffect(() => {
    // We update the text/icon only when we see a change to a non-pending status
    // Pending statuses keep the original text shown (e.g. "Retry" text on failed
    // install shouldn't change to "Install" text because it was clicked and went
    // pending. Once the status is no longer pending, like 'installed' the
    // text will update to "Reinstall")
    if (status !== "pending_install" && status !== "pending_uninstall") {
      setButtonDisplayConfig({
        install: getInstallerActionButtonConfig("install", ui_status),
        uninstall: getInstallerActionButtonConfig("uninstall", ui_status),
      });
    }
  }, [status, ui_status]);

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
      <HostInstallerActionButton
        baseClass={baseClass}
        disabled={
          status === "pending_install" || status === "pending_uninstall"
        }
        onClick={onClickInstallAction}
        text={buttonDisplayConfig.install.text}
        icon={buttonDisplayConfig.install.icon}
        testId={`${baseClass}__install-button--test`}
      />
      {canUninstallSoftware && (
        <HostInstallerActionButton
          baseClass={baseClass}
          disabled={
            status === "pending_install" || status === "pending_uninstall"
          }
          onClick={onClickUninstallAction}
          text={buttonDisplayConfig.uninstall.text}
          icon={buttonDisplayConfig.uninstall.icon}
          testId={`${baseClass}__uninstall-button--test`}
        />
      )}
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
      sortType: installStatusSortType,
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
