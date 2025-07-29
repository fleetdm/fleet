import React, { useEffect, useState } from "react";
import { CellProps, Column } from "react-table";

import {
  IDeviceSoftware,
  IHostSoftware,
  IHostSoftwareWithUiStatus,
  IVPPHostSoftware,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

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
import HostInstallerActionCell, {
  HostInstallerActionButton,
} from "../../HostSoftwareLibrary/HostInstallerActionCell/HostInstallerActionCell";

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
  deviceToken: string;
  onInstallOrUninstall: () => void;
  onShowUpdateDetails: (software: IDeviceSoftware) => void;
  onShowInstallDetails: (hostSoftware: IHostSoftware) => void;
  onShowVPPInstallDetails: (hostSoftware: IVPPHostSoftware) => void;
  onShowUninstallDetails: (details?: ISoftwareUninstallDetails) => void;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (software: IHostSoftwareWithUiStatus) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  deviceToken,
  onInstallOrUninstall,
  onShowUpdateDetails,
  onShowInstallDetails,
  onShowVPPInstallDetails,
  onShowUninstallDetails,
  onClickInstallAction,
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
            isMyDevicePage
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders, generateSoftwareTableData };
