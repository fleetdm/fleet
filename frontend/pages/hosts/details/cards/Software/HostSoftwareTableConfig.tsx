import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";

import {
  formatSoftwareType,
  IHostSoftware,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import {
  HostPlatform,
  isIPadOrIPhone,
  isLinuxLike,
  isMacOS,
  isWindows,
} from "interfaces/platform";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import InstalledPathCell from "pages/SoftwarePage/components/tables/InstalledPathCell";
import HashCell from "pages/SoftwarePage/components/tables/HashCell/HashCell";
import TooltipWrapper from "components/TooltipWrapper";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import VulnerabilitiesCell from "pages/SoftwarePage/components/tables/VulnerabilitiesCell";
import VersionCell from "pages/SoftwarePage/components/tables/VersionCell";
import { getVulnerabilities } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";
import { getAutomaticInstallPoliciesCount } from "pages/SoftwarePage/helpers";
import { sourcesWithLastOpenedTime } from "pages/hosts/details/components/InventoryVersions/InventoryVersions";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IVulnerabilitiesCellProps = IInstalledVersionsCellProps;
type IInstalledPathCellProps = IInstalledVersionsCellProps;

interface ISoftwareTableHeadersProps {
  router: InjectedRouter;
  teamId: number;
  onShowInventoryVersions: (software: IHostSoftware) => void;
  platform: HostPlatform;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  router,
  teamId,
  onShowInventoryVersions,
  platform,
}: ISoftwareTableHeadersProps): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const {
          id,
          name,
          display_name,
          source,
          app_store_app,
          software_package,
          icon_url,
        } = cellProps.row.original;

        const softwareTitleDetailsPath = getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(id.toString()),
          { team_id: teamId }
        );

        const hasInstaller = !!app_store_app || !!software_package;
        const isSelfService =
          app_store_app?.self_service || software_package?.self_service;
        const automaticInstallPoliciesCount = getAutomaticInstallPoliciesCount(
          cellProps.row.original
        );
        const isAndroidPlayStoreApp =
          !!app_store_app && source === "android_apps";

        return (
          <SoftwareNameCell
            name={name}
            display_name={display_name}
            source={source}
            iconUrl={icon_url}
            path={softwareTitleDetailsPath}
            router={router}
            hasInstaller={hasInstaller}
            isSelfService={isSelfService}
            automaticInstallPoliciesCount={automaticInstallPoliciesCount}
            pageContext="hostDetails"
            isIosOrIpadosApp={isIpadOrIphoneSoftwareSource(source)}
            isAndroidPlayStoreApp={isAndroidPlayStoreApp}
          />
        );
      },
    },
    {
      Header: "Installed version",
      id: "version",
      disableSortBy: true,
      // we use function as accessor because we have two columns that
      // need to access the same data. This is not supported with a string
      // accessor.
      accessor: (originalRow) => originalRow.installed_versions,
      Cell: (cellProps: IInstalledVersionsCellProps) => {
        return <VersionCell versions={cellProps.cell.value} />;
      },
    },
    {
      Header: "Type",
      disableSortBy: true,
      id: "source",
      Cell: (cellProps: ITableStringCellProps) => {
        const { source, extension_for } = cellProps.row.original;
        const value = formatSoftwareType({ source, extension_for });
        return <TextCell value={value} />;
      },
    },
    {
      Header: (): JSX.Element => {
        let tooltipContent = <></>;

        if (isMacOS(platform)) {
          tooltipContent = (
            <>When the version installed most recently was last opened.</>
          );
        } else if (isLinuxLike(platform) || isWindows(platform)) {
          tooltipContent = <>When any version was last opened.</>;
        } else if (isIPadOrIPhone(platform)) {
          tooltipContent = <>Date and time of last open.</>;
        }

        const lastOpenedHeader = tooltipContent ? (
          <TooltipWrapper tipContent={tooltipContent}>
            Last opened
          </TooltipWrapper>
        ) : (
          "Last opened"
        );
        return <HeaderCell value={lastOpenedHeader} disableSortBy />;
      },
      id: "Last opened",
      disableSortBy: true,
      accessor: (originalRow) => {
        const { source } = originalRow;
        const versions = originalRow.installed_versions || [];

        // Extract all last_opened_at values that are actual dates (not empty strings)
        const dateStrings = versions
          .map((v) => v.last_opened_at)
          .filter(
            (date): date is string =>
              date !== undefined &&
              date !== null &&
              date !== "" &&
              !isNaN(new Date(date).getTime())
          );

        // If we have actual dates, return the most recent one
        if (dateStrings.length > 0) {
          return dateStrings.reduce((a, b) =>
            new Date(a).getTime() > new Date(b).getTime() ? a : b
          );
        }

        // If source supports last_opened_at, return empty string to indicate "Never"
        // Otherwise return undefined to indicate "Not supported"
        return sourcesWithLastOpenedTime.has(source) ? "" : undefined;
      },
      Cell: (cellProps: ITableStringCellProps) => {
        const { source } = cellProps.row.original;
        const lastOpenedAt = cellProps.cell.value;

        // If we have a non-empty string value, display it
        if (lastOpenedAt && lastOpenedAt !== "") {
          return (
            <TextCell
              value={<HumanTimeDiffWithDateTip timeString={lastOpenedAt} />}
            />
          );
        }

        // If last_opened_at is an empty string, it means the software supports
        // the field but hasn't been opened
        if (lastOpenedAt === "") {
          return <TextCell value="Never" />;
        }

        // If last_opened_at is undefined/missing, check if source supports it
        return sourcesWithLastOpenedTime.has(source) ? (
          <TextCell value="Never" />
        ) : (
          <TextCell value="Not supported" grey />
        );
      },
    },
    {
      Header: "Vulnerabilities",
      accessor: (originalRow) => originalRow.installed_versions,
      disableSortBy: true,
      Cell: (cellProps: IVulnerabilitiesCellProps) => {
        if (isIpadOrIphoneSoftwareSource(cellProps.row.original.source)) {
          return <TextCell value="Not supported" grey />;
        }
        const vulnerabilities = getVulnerabilities(cellProps.cell.value ?? []);
        return <VulnerabilitiesCell vulnerabilities={vulnerabilities} />;
      },
    },
    {
      Header: "File path",
      accessor: (originalRow) => originalRow.installed_versions,
      disableSortBy: true,
      Cell: (cellProps: IInstalledPathCellProps) => {
        if (isIpadOrIphoneSoftwareSource(cellProps.row.original.source)) {
          return <TextCell value="Not supported" grey />;
        }

        const onClickMultiplePaths = () => {
          onShowInventoryVersions(cellProps.row.original);
        };

        return (
          <InstalledPathCell
            installedVersion={cellProps.row.original.installed_versions}
            onClickMultiplePaths={onClickMultiplePaths}
          />
        );
      },
    },
    {
      Header: "Hash",
      accessor: (originalRow) => originalRow.installed_versions,
      disableSortBy: true,
      Cell: (cellProps: IInstalledPathCellProps) => {
        if (isIpadOrIphoneSoftwareSource(cellProps.row.original.source)) {
          return <TextCell value="Not supported" grey />;
        }

        const onClickMultipleHashes = () => {
          onShowInventoryVersions(cellProps.row.original);
        };

        return (
          <HashCell
            installedVersion={cellProps.row.original.installed_versions}
            onClickMultipleHashes={onClickMultipleHashes}
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders };
