import React from "react";
import { CellProps, Column } from "react-table";

import {
  IDeviceSoftware,
  IHostSoftware,
  IHostSoftwareWithUiStatus,
  IVPPHostSoftware,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { ISWUninstallDetailsParentState } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import InstallStatusCell from "../InstallStatusCell/InstallStatusCell";
import { installStatusSortType } from "../helpers";
import HostInstallerActionCell from "../../HostSoftwareLibrary/HostInstallerActionCell/HostInstallerActionCell";

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

export const generateSoftwareTableData = (
  software: IHostSoftwareWithUiStatus[]
): IHostSoftwareWithUiStatus[] => {
  return software;
};

interface ISelfServiceTableHeaders {
  onShowUpdateDetails: (software: IDeviceSoftware) => void;
  onShowInstallDetails: (hostSoftware: IHostSoftware) => void;
  onShowVPPInstallDetails: (hostSoftware: IVPPHostSoftware) => void;
  onShowUninstallDetails: (
    uninstallDetails: ISWUninstallDetailsParentState
  ) => void;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (software: IHostSoftwareWithUiStatus) => void;
  onClickOpenInstructionsAction: (software: IHostSoftwareWithUiStatus) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  onShowUpdateDetails,
  onShowInstallDetails,
  onShowVPPInstallDetails,
  onShowUninstallDetails,
  onClickInstallAction,
  onClickUninstallAction,
  onClickOpenInstructionsAction,
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
        const { name, source, icon_url } = cellProps.row.original;
        return (
          <SoftwareNameCell
            name={name}
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
