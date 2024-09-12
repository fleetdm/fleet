import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IFleetMaintainedApp } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { buildQueryStringFromParams } from "utilities/url";

type IFleetMaintainedAppsTableConfig = Column<IFleetMaintainedApp>;
type ITableStringCellProps = IStringCellProps<IFleetMaintainedApp>;
type ITableHeaderProps = IHeaderProps<IFleetMaintainedApp>;

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

        const path = `/new_path?${buildQueryStringFromParams({
          team_id: teamId,
        })}`;

        return <SoftwareNameCell name={name} path={path} router={router} />;
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "Version",
      accessor: "version",
      Cell: ({ cell }: ITableStringCellProps) => (
        <TextCell value={cell.value} />
      ),
      disableSortBy: true,
    },
    {
      Header: "Platform",
      accessor: "platform",
      Cell: ({ cell }: ITableStringCellProps) => {
        return <span>{cell.value}</span>;
      },
      disableSortBy: true,
    },
  ];
};
