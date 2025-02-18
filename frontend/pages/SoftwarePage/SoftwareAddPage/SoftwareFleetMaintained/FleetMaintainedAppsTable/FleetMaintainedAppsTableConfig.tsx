import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { APPLE_PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { IFleetMaintainedApp } from "interfaces/software";
import { buildQueryStringFromParams } from "utilities/url";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import TooltipWrapper from "components/TooltipWrapper";

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
        const { name, id } = cellProps.row.original;

        const path = `${PATHS.SOFTWARE_FLEET_MAINTAINED_DETAILS(
          id
        )}?${buildQueryStringFromParams({
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
      Header: () => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                Currently, only macOS apps are <br />
                supported.
              </>
            }
          >
            Platform
          </TooltipWrapper>
        );
        return <HeaderCell value={titleWithToolTip} disableSortBy />;
      },
      accessor: "platform",
      Cell: ({ cell }: ITableStringCellProps) => (
        <TextCell
          value={
            APPLE_PLATFORM_DISPLAY_NAMES[
              cell.value as keyof typeof APPLE_PLATFORM_DISPLAY_NAMES
            ]
          }
        />
      ),
      disableSortBy: true,
    },
  ];
};
