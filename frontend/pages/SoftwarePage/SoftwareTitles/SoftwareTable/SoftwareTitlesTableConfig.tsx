import React from "react";
import { CellProps, Column } from "react-table";
import { InjectedRouter } from "react-router";

import {
  ISoftwareTitleVersion,
  ISoftwareTitle,
  formatSoftwareType,
} from "interfaces/software";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";

import VersionCell from "../../components/VersionCell";
import VulnerabilitiesCell from "../../components/VulnerabilitiesCell";
import SoftwareIcon from "../../components/icons/SoftwareIcon";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties

type ISoftwareTitlesTableConfig = Column<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;
type IVersionsCellProps = CellProps<ISoftwareTitle, ISoftwareTitle["versions"]>;
type IVulnerabilitiesCellProps = IVersionsCellProps;
type IHostCountCellProps = CellProps<
  ISoftwareTitle,
  ISoftwareTitle["hosts_count"]
>;
type IViewAllHostsLinkProps = CellProps<ISoftwareTitle>;

type ITableHeaderProps = IHeaderProps<ISoftwareTitle>;

const getVulnerabilities = (versions: ISoftwareTitleVersion[]) => {
  if (!versions) {
    return [];
  }
  const vulnerabilities = versions.reduce((acc: string[], currentVersion) => {
    if (
      currentVersion.vulnerabilities &&
      currentVersion.vulnerabilities.length !== 0
    ) {
      acc.push(...currentVersion.vulnerabilities);
    }
    return acc;
  }, []);
  return vulnerabilities;
};

const generateTableHeaders = (
  router: InjectedRouter,
  teamId?: number
): ISoftwareTitlesTableConfig[] => {
  const softwareTableHeaders: ISoftwareTitlesTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      disableSortBy: false,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        const { id, name, source } = cellProps.row.original;

        const teamQueryParam = buildQueryStringFromParams({ team_id: teamId });
        const softwareTitleDetailsPath = `${PATHS.SOFTWARE_TITLE_DETAILS(
          id.toString()
        )}?${teamQueryParam}`;

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();

          router?.push(softwareTitleDetailsPath);
        };

        return (
          <LinkCell
            path={softwareTitleDetailsPath}
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
      accessor: "versions",
      Cell: (cellProps: IVersionsCellProps) => (
        <VersionCell versions={cellProps.cell.value} />
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
    // the "vulnerabilities" accessor is used but the data is actually coming
    // from the version attribute. We do this as we already have a "versions"
    // attribute used for the "Version" column and we cannot reuse. This is a
    // limitation of react-table.
    // With the versions data, we can sum up the vulnerabilities to get the
    // total number of vulnerabilities for the software title
    {
      Header: "Vulnerabilities",
      disableSortBy: true,
      Cell: (cellProps: IVulnerabilitiesCellProps) => {
        const vulnerabilities = getVulnerabilities(
          cellProps.row.original.versions ?? []
        );
        return <VulnerabilitiesCell vulnerabilities={vulnerabilities} />;
      },
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
      disableSortBy: true,
      Cell: (cellProps: IViewAllHostsLinkProps) => {
        return (
          <ViewAllHostsLink
            queryParams={{
              software_title_id: cellProps.row.original.id,
              team_id: teamId, // TODO: do we need team id here?
            }}
            className="software-link"
            rowHover
          />
        );
      },
    },
  ];

  return softwareTableHeaders;
};

export default generateTableHeaders;
