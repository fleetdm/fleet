import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";

import {
  IHostSoftware,
  SoftwareSource,
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import InstalledPathCell from "pages/SoftwarePage/components/tables/InstalledPathCell";
import HashCell from "pages/SoftwarePage/components/tables/HashCell/HashCell";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import VulnerabilitiesCell from "pages/SoftwarePage/components/tables/VulnerabilitiesCell";
import VersionCell from "pages/SoftwarePage/components/tables/VersionCell";
import { getVulnerabilities } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/SoftwareTitlesTableConfig";

export const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "showDetails", label: "Show details", disabled: false },
  { value: "install", label: "Install", disabled: false },
  { value: "uninstall", label: "Uninstall", disabled: false },
];

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
  onClickMoreDetails: (software: IHostSoftware) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  router,
  teamId,
  onClickMoreDetails,
}: ISoftwareTableHeadersProps): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { id, name, source, app_store_app } = cellProps.row.original;

        const softwareTitleDetailsPath = getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(id.toString()),
          { team_id: teamId }
        );

        return (
          <SoftwareNameCell
            name={name}
            source={source}
            iconUrl={app_store_app?.icon_url}
            path={softwareTitleDetailsPath}
            router={router}
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
      accessor: "source",
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell
          value={cellProps.cell.value}
          formatter={() =>
            formatSoftwareType({
              source: cellProps.cell.value as SoftwareSource,
            })
          }
        />
      ),
    },
    {
      Header: "Last used",
      disableSortBy: true,
      accessor: (originalRow) => {
        // Extract all last_opened_at values, filter out null/undefined, and ensure valid dates
        const dateStrings = (originalRow.installed_versions || [])
          .map((v) => v.last_opened_at)
          .filter(
            (date): date is string => !!date && !isNaN(new Date(date).getTime())
          );

        if (dateStrings.length === 0) return null;

        // Find the most recent date string by comparing their Date values
        const mostRecent = dateStrings.reduce((a, b) =>
          new Date(a).getTime() > new Date(b).getTime() ? a : b
        );

        return mostRecent; // cellProps.cell.value = mostRecent;
      },
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <TextCell
            value={
              cellProps.cell.value ? (
                <HumanTimeDiffWithDateTip timeString={cellProps.cell.value} />
              ) : (
                DEFAULT_EMPTY_CELL_VALUE
              )
            }
            grey={!cellProps.cell.value}
          />
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
          onClickMoreDetails(cellProps.row.original);
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
          onClickMoreDetails(cellProps.row.original);
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
