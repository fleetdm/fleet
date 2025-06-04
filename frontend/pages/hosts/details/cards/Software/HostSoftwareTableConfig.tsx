import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";
import { cloneDeep } from "lodash";

import {
  IHostAppStoreApp,
  IHostSoftware,
  IHostSoftwarePackage,
  SoftwareInstallStatus,
  SoftwareSource,
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import {
  IHeaderProps,
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { dateAgo } from "utilities/date_format";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import InstalledPathCell from "pages/SoftwarePage/components/tables/InstalledPathCell";
import HashCell from "pages/SoftwarePage/components/tables/HashCell/HashCell";
import TooltipWrapper from "components/TooltipWrapper";

import VulnerabilitiesCell from "pages/SoftwarePage/components/tables/VulnerabilitiesCell";
import VersionCell from "pages/SoftwarePage/components/tables/VersionCell";
import { getVulnerabilities } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/SoftwareTitlesTableConfig";

import { getDropdownOptionTooltipContent } from "../../HostDetailsPage/HostActionsDropdown/helpers";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

export const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "showDetails", label: "Show details", disabled: false },
  { value: "install", label: "Install", disabled: false },
  { value: "uninstall", label: "Uninstall", disabled: false },
];

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableNumberCellProps = INumberCellProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IVulnerabilitiesCellProps = IInstalledVersionsCellProps;
type IInstalledPathCellProps = IInstalledVersionsCellProps;

export interface generateActionsProps {
  userHasSWWritePermission: boolean;
  hostScriptsEnabled: boolean;
  hostCanWriteSoftware: boolean;
  softwareIdActionPending: number | null;
  softwareId: number;
  status: SoftwareInstallStatus | null;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  hostMDMEnrolled?: boolean;
}

export const generateActions = ({
  userHasSWWritePermission,
  hostScriptsEnabled,
  softwareIdActionPending,
  softwareId,
  status,
  software_package,
  app_store_app,
  hostMDMEnrolled,
}: generateActionsProps) => {
  // this gives us a clean slate of the default actions so we can modify
  // the options.
  const actions = cloneDeep(DEFAULT_ACTION_OPTIONS);

  // we want to hide the install/uninstall actions if (1) this item doesn't have a
  // software_package or app_store_app or (2) the user doens't have write permission
  const hideActions =
    (!app_store_app && !software_package) || !userHasSWWritePermission;

  const indexInstallAction = actions.findIndex((a) => a.value === "install");
  if (indexInstallAction === -1) {
    // this should never happen unless the default actions change, but if it does we'll throw an
    // error to fail loudly so that we know to update this function
    throw new Error("Install action not found in default actions");
  }
  const indexUninstallAction = actions.findIndex(
    (a) => a.value === "uninstall"
  );
  if (indexUninstallAction === -1) {
    // this should never happen unless the default actions change, but if it does we'll throw an
    // error to fail loudly so that we know to update this function
    throw new Error("Uninstall action not found in default actions");
  }

  if (indexInstallAction > indexUninstallAction) {
    // subsquent code depends on relative index order; this shouldn't change, but if it does we'll throw an
    // error to fail loudly so that we know to update this function
    throw new Error("Order of install/uninstall actions changed");
  }

  if (hideActions) {
    // Reverse order to not change index of subsequent array element before removal
    actions.splice(indexUninstallAction, 1);
    actions.splice(indexInstallAction, 1);
  } else {
    // if host's scripts are disabled, and this isn't a VPP app, disable
    // install/uninstall with tooltip
    if (!hostScriptsEnabled && !app_store_app) {
      actions[indexInstallAction].disabled = true;
      actions[indexUninstallAction].disabled = true;

      actions[
        indexInstallAction
      ].tooltipContent = getDropdownOptionTooltipContent("installSoftware");
      actions[
        indexUninstallAction
      ].tooltipContent = getDropdownOptionTooltipContent("uninstallSoftware");
    }

    // user has software write permission for host
    const pendingStatuses = ["pending_install", "pending_uninstall"];

    // if locally pending (waiting for API response) or pending install/uninstall,
    // disable both install and uninstall
    if (
      softwareId === softwareIdActionPending ||
      pendingStatuses.includes(status || "")
    ) {
      actions[indexInstallAction].disabled = true;
      actions[indexUninstallAction].disabled = true;
    }
  }

  if (app_store_app) {
    // remove uninstall for VPP apps
    actions.splice(indexUninstallAction, 1);

    if (!hostMDMEnrolled) {
      actions[indexInstallAction].disabled = true;
      actions[indexInstallAction].tooltipContent =
        "To install, turn on MDM for this host.";
    }
  }
  return actions;
};

interface ISoftwareTableHeadersProps {
  userHasSWWritePermission: boolean;
  hostScriptsEnabled?: boolean;
  hostCanWriteSoftware: boolean;
  softwareIdActionPending: number | null;
  router: InjectedRouter;
  teamId: number;
  hostMDMEnrolled?: boolean;
  onClickMoreDetails: (software: IHostSoftware) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  userHasSWWritePermission,
  hostScriptsEnabled = false,
  hostCanWriteSoftware,
  softwareIdActionPending,
  router,
  teamId,
  hostMDMEnrolled,
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
