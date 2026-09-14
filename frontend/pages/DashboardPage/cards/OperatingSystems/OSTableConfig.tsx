/**
 dashboard/:osname Dashboard > OS dropdown selected > Operating system table
 software/os > OS tab > Operating system table
*/

import React from "react";
import { CellProps, Column } from "react-table";
import { InjectedRouter } from "react-router";

import { getPathWithQueryParams } from "utilities/url";
import PATHS from "router/paths";
import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";
import {
  ISoftwareVulnerability,
  ROLLING_ARCH_LINUX_NAMES,
} from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";

import VulnerabilitiesCell from "pages/SoftwarePage/components/tables/VulnerabilitiesCell";
import OSIcon from "pages/SoftwarePage/components/icons/OSIcon";
import {
  IHeaderProps,
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { isVulnUnsupportedPlatform } from "interfaces/platform";
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

type IHostHeaderProps = IHeaderProps<IOperatingSystemVersion>;

// Windows feature-update codename, e.g. "21H2", "23H1" — one of the two
// documented shapes of fleet.OSVersion.Version (the other being
// dot-separated numbers). Treated as a [year, half] version for comparison.
const WINDOWS_FEATURE_UPDATE_PATTERN = /^(\d{2})H([12])$/;
// Matches server/service/hosts.go's numericVersionPattern. Deliberately
// stricter than `Number()` coercion, which also accepts "", "1.", ".1",
// "+1", "1e2", and hex/whitespace as valid numbers.
const NUMERIC_VERSION_PATTERN = /^\d+(\.\d+)*$/;
// Ubuntu LTS releases report a literal " LTS" suffix in fleet.OSVersion.Version
// (e.g. "22.04.9 LTS") — osquery's os_version.version column reports it that
// way and Fleet stores it verbatim. Matches server/service/hosts.go's
// ubuntuLTSSuffixPattern.
const UBUNTU_LTS_SUFFIX_PATTERN = /\s+LTS$/i;

/** Splits an OS version string into numeric segments for comparison, or
 * returns null if it doesn't match a documented shape. Deliberately NOT the
 * shared frontend/utilities/helpers.tsx `compareVersions` — that helper is
 * intentionally kept simple for software versions (which can have messy
 * suffixes but never Windows codenames), while OS versions need codename
 * support and are usually one of a few clean shapes (non-comparable formats
 * like Arch Linux's "rolling" fall through to null here). Mirrors
 * server/service/hosts.go's versionSegments/compareOSVersions, which faces
 * the same shapes server-side for the Software > OS page's sort. */
const osVersionSegments = (version: string): number[] | null => {
  const windowsMatch = version.match(WINDOWS_FEATURE_UPDATE_PATTERN);
  if (windowsMatch) {
    return [Number(windowsMatch[1]), Number(windowsMatch[2])];
  }
  const withoutLTSSuffix = version.replace(UBUNTU_LTS_SUFFIX_PATTERN, "");
  if (!NUMERIC_VERSION_PATTERN.test(withoutLTSSuffix)) {
    return null;
  }
  return withoutLTSSuffix.split(".").map(Number);
};

/** Compares two OS version strings numerically by segment (so "26.10" >
 * "26.6") or, for Windows feature-update codenames, by year and half (so
 * "22H1" > "21H2") — rather than as plain strings. A version matching
 * neither shape (e.g. Arch Linux's "rolling") isn't comparable this way, so
 * it sorts before any version that is. */
export const compareOSVersionStrings = (
  version1: string,
  version2: string
): number => {
  const v1Parts = osVersionSegments(version1);
  const v2Parts = osVersionSegments(version2);

  if (!v1Parts && !v2Parts) return 0;
  if (!v1Parts) return -1;
  if (!v2Parts) return 1;

  const maxLength = Math.max(v1Parts.length, v2Parts.length);
  for (let i = 0; i < maxLength; i += 1) {
    const v1Part = v1Parts[i] || 0;
    const v2Part = v2Parts[i] || 0;
    if (v1Part !== v2Part) return v1Part < v2Part ? -1 : 1;
  }
  return 0;
};

interface IOSTableConfigOptions {
  includeName?: boolean;
  includeVulnerabilities?: boolean;
  includeIcon?: boolean;
  /** Sum of hosts_count per platform *within the rows currently loaded*
   * (the dashboard card fetches without page/per_page, so the API defaults
   * to the first 20 results — not necessarily every platform's fleet-wide
   * total, if there are more than 20 distinct OS versions) — used to group
   * the Version column's sort by platform (most hosts first) before
   * ordering by version within each platform, since comparing versions
   * across platforms isn't meaningful (e.g. macOS "26.6" vs. Windows
   * "22H1"). Only used for client-side sorting (the dashboard card); the
   * server-driven Software > OS table ignores this column's sortType
   * entirely. */
  platformHostTotals?: Record<string, number>;
}

/** Orders the Version column by platform group (most hosts first, always
 * — this part doesn't flip with `desc`, since counteracting react-table's
 * blanket sign flip when sorted descending keeps the group order fixed),
 * then by version within each platform group (which does flip with
 * `desc`, same as any other ascending comparator) — comparing versions
 * across platforms isn't meaningful (e.g. macOS "26.6" vs. Windows
 * "22H1"). If two platforms tie on host total (including both being
 * absent from `platformHostTotals`), falls back to comparing platform
 * names so rows still group together instead of relying on the sort
 * being stable. */
export const compareOSTableVersions = (
  rowA: IOperatingSystemVersion,
  rowB: IOperatingSystemVersion,
  desc: boolean,
  platformHostTotals: Record<string, number> = {}
): number => {
  if (rowA.platform !== rowB.platform) {
    const totalA = platformHostTotals[rowA.platform] ?? 0;
    const totalB = platformHostTotals[rowB.platform] ?? 0;
    // Fixed (direction-invariant) target: most hosts first, or
    // alphabetical by platform name if host totals tie.
    const target =
      totalA === totalB
        ? rowA.platform.localeCompare(rowB.platform)
        : totalB - totalA;
    return desc ? -target : target;
  }
  return compareOSVersionStrings(rowA.version, rowB.version);
};

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
        { fleet_id: teamId }
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
    Header: (cellProps: IHostHeaderProps) => (
      <HeaderCell
        value="Version"
        disableSortBy={false}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    disableSortBy: false,
    accessor: "version",
    sortType: (rowA, rowB, _columnId, desc) =>
      compareOSTableVersions(
        rowA.original,
        rowB.original,
        !!desc,
        configOptions?.platformHostTotals
      ),
    Cell: (cellProps: IVersionCellProps) => {
      const { version, name_only } = cellProps.row.original;
      if (
        ROLLING_ARCH_LINUX_NAMES.includes(name_only) &&
        version === "rolling"
      ) {
        return (
          <TextCell value={<TooltipWrapperArchLinuxRolling capitalized />} />
        );
      }
      return <TextCell value={version} />;
    },
  },
  {
    Header: "Vulnerabilities",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps) => {
      const platform = cellProps.row.original.platform;
      if (isVulnUnsupportedPlatform(platform)) {
        return (
          <TooltipWrapper
            tipContent={
              <>
                Vulnerabilities are currently supported on
                <br />
                macOS, Windows, Linux, and Android.{" "}
                <CustomLink
                  url="https://fleetdm.com/guides/vulnerability-processing#coverage"
                  variant="tooltip-link"
                  text="Learn more"
                  newTab
                />
              </>
            }
            position="top"
            underline={false}
            showArrow
            fixedPositionStrategy
          >
            <TextCell value="Not supported" grey />
          </TooltipWrapper>
        );
      }
      return (
        <VulnerabilitiesCell
          vulnerabilities={cellProps.cell.value}
          vulnerabilitiesCount={cellProps.row.original.vulnerabilities_count}
        />
      );
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
            fleet_id: teamId,
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
