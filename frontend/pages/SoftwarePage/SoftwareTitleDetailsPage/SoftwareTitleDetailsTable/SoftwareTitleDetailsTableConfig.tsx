import React from "react";
import { InjectedRouter } from "react-router";

import {
  ISoftwareTitleVersion,
  ISoftwareVulnerability,
} from "interfaces/software";
import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";

import VulnerabilitiesCell from "../../components/VulnerabilitiesCell";

interface ISoftwareTitleDetailsTableConfigProps {
  router: InjectedRouter;
  teamId?: number;
}
interface ICellProps {
  cell: {
    value: number | string | ISoftwareVulnerability[];
  };
  row: {
    original: ISoftwareTitleVersion;
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

const generateSoftwareTitleDetailsTableConfig = ({
  router,
  teamId,
}: ISoftwareTitleDetailsTableConfigProps) => {
  const tableHeaders = [
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps: IVersionCellProps): JSX.Element => {
        const { id } = cellProps.row.original;
        const teamQueryParam = buildQueryStringFromParams({ team_id: teamId });
        const softwareVersionDetailsPath = `${PATHS.SOFTWARE_VERSION_DETAILS(
          id.toString()
        )}?${teamQueryParam}`;

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();
          router?.push(softwareVersionDetailsPath);
        };

        // TODO: make only text clickable
        return (
          <LinkCell
            className="name-link"
            path={softwareVersionDetailsPath}
            customOnClick={onClickSoftware}
            value={cellProps.cell.value}
          />
        );
      },
    },
    {
      title: "Vulnerabilities",
      Header: "Vulnerabilities",
      disableSortBy: true,
      // the "vulnerabilities" accessor is used but the data is actually coming
      // from the version attribute. We do this as we already have a "versions"
      // attribute used for the "Version" column and we cannot reuse. This is a
      // limitation of react-table.
      // With the versions data, we can sum up the vulnerabilities to get the
      // total number of vulnerabilities for the software title
      accessor: "vulnerabilities",
      Cell: (cellProps: IVulnCellProps): JSX.Element => (
        <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />
        // TODO: tooltip
      ),
    },
    {
      title: "Hosts",
      Header: "Hosts",
      disableSortBy: true,
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
                  team_id: teamId,
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

  return tableHeaders;
};

export default generateSoftwareTitleDetailsTableConfig;
