/**
 dashboard/:osname Dashboard > OS dropdown selected > Operating system table
 software/os > OS tab > Operating system table
*/

import React from "react";
import { CellProps, Column, HeaderProps } from "react-table";
import { InjectedRouter } from "react-router";

import { getPathWithQueryParams } from "utilities/url";
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
import TooltipWrapper from "components/TooltipWrapper";

import VulnerabilitiesCell from "pages/SoftwarePage/components/tables/VulnerabilitiesCell";
import OSIcon from "pages/SoftwarePage/components/icons/OSIcon";
import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { isLinuxLike } from "interfaces/platform";
import TooltipWrapperArchLinuxRolling from "components/TooltipWrapperArchLinuxRolling";

type ITableColumnConfig = Column<IOperatingSystemVersion>;

type INameCellProps = IStringCellProps<IOperatingSystemVersion>;
type IVersionCellProps = IStringCellProps<IOperatingSystemVersion>;
type IVulnCellProps = CellProps<
  IOperatingSystemVersion,
  ISoftwareVulnerability[]
>;
type IHostCountCellProps = INumberCellProps<IOperatingSystemVersion>;
type IViewAllHostsLinkProps = CellProps<IOperatingSystemVersion>;

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

      const { name_only, os_version_id, platform } = cellProps.row.original;

      const softwareOsDetailsPath = getPathWithQueryParams(
        PATHS.SOFTWARE_OS_DETAILS(os_version_id),
        { team_id: teamId }
      );

      const onClickSoftware = (e: React.MouseEvent) => {
        // Allows for button to be clickable in a clickable row
        e.stopPropagation();

        router?.push(softwareOsDetailsPath);
      };

      return (
        <LinkCell
          path={softwareOsDetailsPath}
          customOnClick={onClickSoftware}
          tooltipTruncate
          prefix={<OSIcon name={platform} />}
          value={name_only}
        />
      );
    },
  },
  {
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: IVersionCellProps) => {
      const value = cellProps.cell.value;
      if (
        (cellProps.row.values.name_only === "Arch Linux" ||
          cellProps.row.values.name_only === "Arch Linux ARM") &&
        value === "rolling"
      ) {
        return <TextCell value={<TooltipWrapperArchLinuxRolling />} />;
      }
      return <TextCell value={value} />;
    },
  },
  {
    Header: (): JSX.Element => {
      const titleWithTooltip = (
        <TooltipWrapper
          tipContent={
            <>
              Vulnerabilities on Linux are currently supported <br />
              for Ubuntu, Debian, and Amazon Linux.
            </>
          }
        >
          Vulnerabilities
        </TooltipWrapper>
      );
      return (
        <>
          <HeaderCell value={titleWithTooltip} disableSortBy />
        </>
      );
    },
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps) => {
      const platform = cellProps.row.original.platform;
      if (
        platform !== "darwin" &&
        platform !== "windows" &&
        !isLinuxLike(platform)
      ) {
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
      const { hosts_count } = cellProps.row.original;
      return (
        <span className="hosts-cell__count">
          <TextCell value={hosts_count} />
        </span>
      );
    },
  },
  {
    Header: "",
    id: "view-all-hosts",
    disableSortBy: true,
    Cell: (cellProps: IViewAllHostsLinkProps) => {
      const { os_version_id } = cellProps.row.original;
      return (
        <ViewAllHostsLink
          queryParams={{
            os_version_id,
            team_id: teamId,
          }}
          className="os-hosts-link"
          rowHover
        />
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
