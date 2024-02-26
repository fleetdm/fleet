import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";

import { buildQueryStringFromParams } from "utilities/url";
import {
  formatSoftwareType,
  ISoftwareVersion,
  ISoftwareVulnerability,
} from "interfaces/software";
import PATHS from "router/paths";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import VulnerabilitiesCell from "../../components/VulnerabilitiesCell";
import SoftwareIcon from "../../components/icons/SoftwareIcon";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: number | string | ISoftwareVulnerability[];
  };
  row: {
    original: ISoftwareVersion;
  };
}
interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface IVersionCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface INumberCellProps extends ICellProps {
  cell: {
    value: number;
  };
}

interface IVulnCellProps extends ICellProps {
  cell: {
    value: ISoftwareVulnerability[];
  };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

const generateTableHeaders = (
  router: InjectedRouter,
  teamId?: number
): Column[] => {
  const softwareTableHeaders = [
    {
      title: "Name",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "name",
      Cell: (cellProps: IStringCellProps): JSX.Element => {
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
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps: IVersionCellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Type",
      Header: "Type",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={formatSoftwareType(cellProps.row.original)} />
      ),
    },
    {
      title: "Vulnerabilities",
      Header: "Vulnerabilities",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Cell: (cellProps: IVulnCellProps): JSX.Element => (
        <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />
      ),
    },
    {
      title: "Hosts",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={cellProps.column.title}
          disableSortBy={false}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "hosts_count",
      Cell: (cellProps: INumberCellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => {
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
