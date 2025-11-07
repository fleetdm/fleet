import React from "react";
import { CellProps, Column } from "react-table";

import {
  IDeviceSoftware,
  IDeviceSoftwareWithUiStatus,
  IHostSoftware,
  IVPPHostSoftware,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { ISWUninstallDetailsParentState } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import InstallStatusCell from "../../../InstallStatusCell/InstallStatusCell";
import { installStatusSortType } from "../../../helpers";
import HostInstallerActionCell from "../../../../HostSoftwareLibrary/HostInstallerActionCell/HostInstallerActionCell";

type ISelfServiceTableConfig = Column<IDeviceSoftwareWithUiStatus>;
type ITableHeaderProps = IHeaderProps<IDeviceSoftwareWithUiStatus>;
type ITableStringCellProps = IStringCellProps<IDeviceSoftwareWithUiStatus>;
type IStatusCellProps = CellProps<
  IDeviceSoftwareWithUiStatus,
  IDeviceSoftwareWithUiStatus["ui_status"]
>;
type IActionCellProps = CellProps<
  IDeviceSoftwareWithUiStatus,
  IDeviceSoftwareWithUiStatus["status"]
>;

const baseClass = "self-service-table";

export const generateSoftwareTableData = (
  software: IDeviceSoftwareWithUiStatus[]
): IDeviceSoftwareWithUiStatus[] => {
  return software;
};

interface ISelfServiceTableHeaders {
  onShowUpdateDetails: (software: IDeviceSoftware) => void;
  onShowInstallDetails: (hostSoftware: IHostSoftware) => void;
  onShowIpaInstallDetails: (hostSoftware: IHostSoftware) => void;
  onShowScriptDetails: (hostSoftware: IHostSoftware) => void;
  onShowVPPInstallDetails: (hostSoftware: IVPPHostSoftware) => void;
  onShowUninstallDetails: (
    uninstallDetails: ISWUninstallDetailsParentState
  ) => void;
  onClickInstallAction: (softwareId: number, isScriptPackage?: boolean) => void;
  onClickUninstallAction: (software: IDeviceSoftwareWithUiStatus) => void;
  onClickOpenInstructionsAction: (
    software: IDeviceSoftwareWithUiStatus
  ) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  onShowUpdateDetails,
  onShowInstallDetails,
  onShowIpaInstallDetails,
  onShowScriptDetails,
  onShowVPPInstallDetails,
  onShowUninstallDetails,
  onClickInstallAction,
  onClickUninstallAction,
  onClickOpenInstructionsAction,
}: ISelfServiceTableHeaders): ISelfServiceTableConfig[] => {
  const tableHeaders: ISelfServiceTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      disableGlobalFilter: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, display_name, source, icon_url } = cellProps.row.original;
        return (
          <SoftwareNameCell
            name={name}
            display_name={display_name}
            source={source}
            iconUrl={icon_url}
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
        <InstallStatusCell
          software={cellProps.row.original}
          onShowUpdateDetails={onShowUpdateDetails}
          onShowInstallDetails={onShowInstallDetails}
          onShowIpaInstallDetails={onShowIpaInstallDetails}
          onShowScriptDetails={onShowScriptDetails}
          onShowVPPInstallDetails={onShowVPPInstallDetails}
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
          <HostInstallerActionCell
            software={cellProps.row.original}
            baseClass={baseClass}
            onClickInstallAction={onClickInstallAction}
            onClickUninstallAction={() =>
              onClickUninstallAction(cellProps.row.original)
            }
            onClickOpenInstructionsAction={() =>
              onClickOpenInstructionsAction(cellProps.row.original)
            }
            isMyDevicePage
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders, generateSoftwareTableData };
