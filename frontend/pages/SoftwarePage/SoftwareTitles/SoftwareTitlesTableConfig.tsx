import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";

import {
  ISoftwareTitleVersion,
  ISoftwareTitle,
  formatSoftwareType,
} from "interfaces/software";
import PATHS from "router/paths";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import VersionCell from "../components/VersionCell";
import VulnerabilitiesCell from "../components/VulnerabilitiesCell";
import SoftwareIcon from "../components/icons/SoftwareIcon";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: number | string | ISoftwareTitleVersion[];
  };
  row: {
    original: ISoftwareTitle;
  };
}
interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface IVersionCellProps extends ICellProps {
  cell: {
    value: ISoftwareTitleVersion[];
  };
  row: {
    original: ISoftwareTitle;
  };
}

interface INumberCellProps extends ICellProps {
  cell: {
    value: number;
  };
}

interface IVulnCellProps extends ICellProps {
  cell: {
    value: ISoftwareTitleVersion[];
  };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

const getVulnerabilities = (versions: ISoftwareTitleVersion[]) => {
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

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();

          router?.push(PATHS.SOFTWARE_TITLE_DETAILS(id.toString()));
        };

        return (
          <LinkCell
            path={PATHS.SOFTWARE_TITLE_DETAILS(id.toString())}
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
      accessor: "versions",
      Cell: (cellProps: IVersionCellProps): JSX.Element => (
        <VersionCell versions={cellProps.cell.value} />
      ),
    },
    {
      title: "Type",
      Header: "Type",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: IStringCellProps): JSX.Element => (
        <TextCell formatter={formatSoftwareType} value={cellProps.cell.value} />
      ),
    },
    // the "vulnerabilities" accessor is used but the data is actually coming
    // from the version attribute. We do this as we already have a "versions"
    // attribute used for the "Version" column and we cannot reuse. This is a
    // limitation of react-table.
    // With the versions data, we can sum up the vulnerabilities to get the
    // total number of vulnerabilities for the software title
    {
      title: "Vulnerabilities",
      Header: "Vulnerabilities",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Cell: (cellProps: IVulnCellProps): JSX.Element => {
        const vulnerabilities = getVulnerabilities(
          cellProps.row.original.versions
        );
        return <VulnerabilitiesCell vulnerabilities={vulnerabilities} />;
      },
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
        <span className="hosts-cell__wrapper">
          <span className="hosts-cell__count">
            <TextCell value={cellProps.cell.value} />
          </span>
          <span className="hosts-cell__link">
            <ViewAllHostsLink
              queryParams={{
                software_title_id: cellProps.row.original.id,
                team_id: teamId, // TODO: do we need team id here?
              }}
              className="software-link"
            />
          </span>
        </span>
      ),
    },
  ];

  return softwareTableHeaders;
};

export default generateTableHeaders;
