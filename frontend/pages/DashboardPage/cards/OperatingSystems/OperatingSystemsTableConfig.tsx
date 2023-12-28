import React from "react";
import { Column } from "react-table";

import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import VulnerabilitiesCell from "pages/SoftwarePage/components/VulnerabilitiesCell";
import { ISoftwareVulnerability } from "interfaces/software";

interface ICellProps {
  cell: {
    value: number | string | ISoftwareVulnerability[];
  };
  row: {
    original: IOperatingSystemVersion;
  };
}

interface IStringCellProps extends ICellProps {
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

const generateDefaultTableHeaders = (teamId?: number): Column[] => [
  {
    Header: "Name",
    disableSortBy: true,
    accessor: "name_only",
    Cell: ({ cell: { value } }: IStringCellProps) => (
      <TextCell
        value={value}
        formatter={(name) => formatOperatingSystemDisplayName(name)}
      />
    ),
  },
  {
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: IStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    Header: "Vulnerabilities",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps): JSX.Element => {
      return <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />;
    },
  },
  {
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: INumberCellProps): JSX.Element => {
      const { hosts_count, name_only, version } = cellProps.row.original;
      return (
        <span className="hosts-cell__wrapper">
          <span className="hosts-cell__count">
            <TextCell value={hosts_count} />
          </span>
          <span className="hosts-cell__link">
            <ViewAllHostsLink
              queryParams={{
                os_name: name_only,
                os_version: version,
                team_id: teamId,
              }}
              className="os-hosts-link"
            />
          </span>
        </span>
      );
    },
  },
];

interface IOSTableConfigOptions {
  includeName?: boolean;
  includeVulnerabilities?: boolean;
  includeIcon?: boolean;
}

const generateTableHeaders = (
  teamId?: number,
  configOptions?: IOSTableConfigOptions
): Column[] => {
  let tableConfig = generateDefaultTableHeaders(teamId);

  if (!configOptions?.includeName) {
    tableConfig = tableConfig.filter(
      (column) => column.accessor !== "name_only"
    );
  }

  if (!configOptions?.includeVulnerabilities) {
    tableConfig = tableConfig.filter(
      (column) => column.accessor !== "vulnerabilities"
    );
  }

  return tableConfig;
};

export default generateTableHeaders;
