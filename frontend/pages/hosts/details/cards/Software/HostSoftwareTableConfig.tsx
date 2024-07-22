import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";
import { cloneDeep } from "lodash";

import {
  IHostSoftware,
  SoftwareInstallStatus,
  formatSoftwareType,
} from "interfaces/software";
import {
  IHeaderProps,
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";

import VulnerabilitiesCell from "pages/SoftwarePage/components/VulnerabilitiesCell";
import VersionCell from "pages/SoftwarePage/components/VersionCell";
import { getVulnerabilities } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/SoftwareTitlesTableConfig";

import InstallStatusCell from "./InstallStatusCell";

const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "showDetails", label: "Show details", disabled: false },
  { value: "install", label: "Install", disabled: false },
];

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableNumberCellProps = INumberCellProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledStatusCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["status"]
>;
type IInstalledVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IVulnerabilitiesCellProps = IInstalledVersionsCellProps;
// type IActionsCellProps = CellProps<IHostSoftware, IHostSoftware["id"]>;

const generateActions = ({
  canInstall,
  installingSoftwareId,
  isFleetdHost,
  softwareId,
  status,
  hasSoftwareToInstall,
}: {
  canInstall: boolean;
  installingSoftwareId: number | null;
  isFleetdHost: boolean;
  softwareId: number;
  status: SoftwareInstallStatus | null;
  hasSoftwareToInstall?: boolean;
}) => {
  // this gives us a clean slate of the default actions so we can modify
  // the options.
  const actions = cloneDeep(DEFAULT_ACTION_OPTIONS);

  const indexInstallAction = actions.findIndex((a) => a.value === "install");
  if (indexInstallAction === -1) {
    // this should never happen unless the default actions change, but if it does we'll throw an
    // error to fail loudly so that we know to update this function
    throw new Error("Install action not found in default actions");
  }

  // remove install if there is no package to install
  if (!hasSoftwareToInstall || !canInstall) {
    actions.splice(indexInstallAction, 1);
    return actions;
  }

  // disable install option if not a fleetd host
  if (!isFleetdHost) {
    actions[indexInstallAction].disabled = true;
    actions[indexInstallAction].tooltipContent =
      "To install software on this host, deploy the fleetd agent with --enable-scripts and refetch host vitals.";
    return actions;
  }

  // disable install option if software is already installing
  if (softwareId === installingSoftwareId || status === "pending") {
    actions[indexInstallAction].disabled = true;
    return actions;
  }

  return actions;
};

interface ISoftwareTableHeadersProps {
  canInstall: boolean;
  installingSoftwareId: number | null;
  isFleetdHost: boolean;
  router: InjectedRouter;
  teamId: number;
  onSelectAction: (software: IHostSoftware, action: string) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  canInstall,
  installingSoftwareId,
  isFleetdHost,
  router,
  teamId,
  onSelectAction,
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

        const softwareTitleDetailsPath = PATHS.SOFTWARE_TITLE_DETAILS(
          id.toString().concat(`?team_id=${teamId}`)
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
      Header: "Install status",
      disableSortBy: true,
      accessor: "status",
      Cell: ({ row: { original } }: IInstalledStatusCellProps) => {
        return <InstallStatusCell {...original} />;
      },
    },
    {
      Header: "Version",
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
          formatter={() => formatSoftwareType({ source: cellProps.cell.value })}
        />
      ),
    },
    {
      Header: "Vulnerabilities",
      accessor: (originalRow) => originalRow.installed_versions,
      disableSortBy: true,
      Cell: (cellProps: IVulnerabilitiesCellProps) => {
        const vulnerabilities = getVulnerabilities(cellProps.cell.value ?? []);
        return <VulnerabilitiesCell vulnerabilities={vulnerabilities} />;
      },
    },
    {
      Header: "",
      disableSortBy: true,
      // the accessor here is insignificant, we just need it as its required
      // but we don't use it.
      accessor: "id",
      Cell: ({ row: { original } }: ITableNumberCellProps) => {
        const {
          id: softwareId,
          status,
          software_package,
          app_store_app,
        } = original;

        return (
          <DropdownCell
            placeholder="Actions"
            options={generateActions({
              canInstall,
              isFleetdHost,
              installingSoftwareId,
              softwareId,
              status,
              hasSoftwareToInstall: !!software_package || !!app_store_app,
            })}
            onChange={(action) => onSelectAction(original, action)}
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders };
