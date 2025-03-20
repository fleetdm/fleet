import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ICombinedFMA } from "interfaces/software";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import InstallerActionCell from "components/TableContainer/DataTable/InstallerActionCell";

type IFleetMaintainedAppsTableConfig = Column<ICombinedFMA>;
type ITableStringCellProps = IStringCellProps<ICombinedFMA>;
type ITableHeaderProps = IHeaderProps<ICombinedFMA>;

// eslint-disable-next-line import/prefer-default-export
export const generateTableConfig = (
  router: InjectedRouter,
  teamId: number
): IFleetMaintainedAppsTableConfig[] => {
  return [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        const { name } = cellProps.row.original;

        return <SoftwareNameCell name={name} />;
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "macOS",
      accessor: "macos",
      Cell: (cellProps: any) => {
        const { macos } = cellProps.row.original;

        return (
          <InstallerActionCell teamId={teamId} value={macos} router={router} />
        );
      },
      disableSortBy: true,
    },
    {
      Header: "Windows",
      accessor: "windows",
      Cell: (cellProps: any) => {
        const { windows } = cellProps.row.original;

        return (
          <InstallerActionCell
            teamId={teamId}
            value={windows}
            router={router}
          />
        );
      },
      disableSortBy: true,
    },
  ];
};
