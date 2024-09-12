/**
 dashboard/:osname Dashboard > OS dropdown selected > Operating system table
 software/os > OS tab > Operating system table
*/

import React from "react";
import { CellProps, Column, HeaderProps } from "react-table";
import { InjectedRouter } from "react-router";

import { buildQueryStringFromParams } from "utilities/url";
import PATHS from "router/paths";
import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";
import { ISoftwareVulnerability } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";

import VulnerabilitiesCell from "pages/SoftwarePage/components/VulnerabilitiesCell";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";

type ITableColumnConfig = Column<IOperatingSystemVersion>;

type INameCellProps = IStringCellProps<IOperatingSystemVersion>;
type IVersionCellProps = IStringCellProps<IOperatingSystemVersion>;
type IVulnCellProps = CellProps<
  IOperatingSystemVersion,
  ISoftwareVulnerability[]
>;
type IHostCountCellProps = INumberCellProps<IOperatingSystemVersion>;
type IHostHeaderProps = HeaderProps<IOperatingSystemVersion>;

interface IOSTableConfigOptions {
  includeName?: boolean;
  includeVulnerabilities?: boolean;
  includeIcon?: boolean;
}

const generateDefaultTableHeaders = (
  teamId?: number,
  router?: InjectedRouter,
  configOptions?: IOSTableConfigOptions
): ITableColumnConfig[] => [
  {
    Header: "Name",
    disableSortBy: true,
    accessor: "name_only",
    Cell: (cellProps: INameCellProps) => {
      if (!configOptions?.includeIcon) {
        return (
          <TextCell
            value={cellProps.cell.value}
            formatter={(name) => formatOperatingSystemDisplayName(name)}
          />
        );
      }

      const { name, os_version_id } = cellProps.row.original;

      const teamQueryParam = buildQueryStringFromParams({
        team_id: teamId,
      });
      const softwareOsDetailsPath = `${PATHS.SOFTWARE_OS_DETAILS(
        os_version_id
      )}?${teamQueryParam}`;

      const onClickSoftware = (e: React.MouseEvent) => {
        // Allows for button to be clickable in a clickable row
        e.stopPropagation();

        router?.push(softwareOsDetailsPath);
      };

      return (
        <LinkCell
          path={softwareOsDetailsPath}
          customOnClick={onClickSoftware}
          value={
            <>
              <SoftwareIcon name={cellProps.row.original.platform} />
              <span className="software-name">{name}</span>
            </>
          }
        />
      );
    },
  },
  {
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: IVersionCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    Header: "Vulnerabilities",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps) => {
      const platform = cellProps.row.original.platform;
      if (platform !== "darwin" && platform !== "windows") {
        return <TextCell value="Not supported" grey />;
      }
      return <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />;
    },
  },
  {
    Header: (cellProps: IHostHeaderProps) => (
      <HeaderCell
        value="Hosts"
        disableSortBy={false}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),

    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: IHostCountCellProps) => {
      const { hosts_count, os_version_id } = cellProps.row.original;
      return (
        <span className="hosts-cell__wrapper">
          <span className="hosts-cell__count">
            <TextCell value={hosts_count} />
          </span>
          <span className="hosts-cell__link">
            <ViewAllHostsLink
              queryParams={{
                os_version_id,
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

// this is also used by frontend/pages/SoftwarePage/SoftwareOS/SoftwareOSTable/SoftwareOSTable.tsx
const generateTableHeaders = (
  teamId?: number,
  router?: InjectedRouter,
  configOptions?: IOSTableConfigOptions
): ITableColumnConfig[] => {
  let tableConfig = generateDefaultTableHeaders(teamId, router, configOptions);

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
