import React from "react";
import { CellProps, Column } from "react-table";
import { InjectedRouter } from "react-router";

import { buildQueryStringFromParams } from "utilities/url";
import {
  formatSoftwareType,
  ISoftwareVersion,
  ISoftwareVulnerability,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import PATHS from "router/paths";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import VulnerabilitiesCell from "../../components/VulnerabilitiesCell";
import SoftwareIcon from "../../components/icons/SoftwareIcon";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties

type ISoftwareVersionsTableConfig = Column<ISoftwareVersion>;
type ITableStringCellProps = IStringCellProps<ISoftwareVersion>;
type IVulnerabilitiesCellProps = CellProps<
  ISoftwareVersion,
  ISoftwareVulnerability[] | null
>;
type IHostCountCellProps = CellProps<ISoftwareVersion, number | undefined>;

type ITableHeaderProps = IHeaderProps<ISoftwareVersion>;

const generateTableHeaders = (
  router: InjectedRouter,
  teamId?: number
): ISoftwareVersionsTableConfig[] => {
  const softwareTableHeaders: ISoftwareVersionsTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      disableSortBy: false,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        const { id, name, source } = cellProps.row.original;

        const teamQueryParam = buildQueryStringFromParams({
          team_id: teamId,
        });
        const softwareVersionDetailsPath = `${PATHS.SOFTWARE_VERSION_DETAILS(
          id.toString()
        )}?${teamQueryParam}`;

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();

          router?.push(softwareVersionDetailsPath);
        };

        return (
          <LinkCell
            path={softwareVersionDetailsPath}
            customOnClick={onClickSoftware}
            value={
              <>
                <SoftwareIcon name={name} source={source} />
                <span className="software-name">{name}</span>
              </>
            }
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      Header: "Type",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={formatSoftwareType(cellProps.row.original)} />
      ),
    },
    {
      Header: "Vulnerabilities",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Cell: (cellProps: IVulnerabilitiesCellProps) => (
        <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />
      ),
    },
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Hosts"
          disableSortBy={false}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "hosts_count",
      Cell: (cellProps: IHostCountCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      Header: "",
      id: "view-all-hosts",
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <>
            {cellProps.row.original && (
              <ViewAllHostsLink
                queryParams={{
                  software_version_id: cellProps.row.original.id,
                  team_id: teamId, // TODO: do we need team id here?
                }}
                className="software-link"
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];

  return softwareTableHeaders;
};

export default generateTableHeaders;
