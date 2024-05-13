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

const generateActions = (
  softwareId: number,
  status: SoftwareInstallStatus | null,
  installingSoftwareId: number | null,
  canInstall: boolean,
  packageToInstall?: string | null
) => {
  // this gives us a clean slate of the default actions so we can modify
  // the options.
  let actions = cloneDeep(DEFAULT_ACTION_OPTIONS);

  // remove install if there is no package to install
  if (!packageToInstall || !canInstall) {
    actions = actions.filter((action) => action.value !== "install");
  }

  // disable install option if software is already installing
  if (softwareId === installingSoftwareId || status === "pending") {
    const installAction = actions.find((action) => action.value === "install");
    if (installAction) {
      installAction.disabled = true;
    }
  }

  return actions;
};

interface ISoftwareTableHeadersProps {
  installingSoftwareId: number | null;
  onSelectAction: (software: IHostSoftware, action: string) => void;
  canInstall: boolean;
  router: InjectedRouter;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  router,
  installingSoftwareId,
  onSelectAction,
  canInstall,
}: ISoftwareTableHeadersProps): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { id, name, source } = cellProps.row.original;

        const softwareTitleDetailsPath = PATHS.SOFTWARE_TITLE_DETAILS(
          id.toString()
        );

        return (
          <SoftwareNameCell
            name={name}
            source={source}
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
      Cell: (cellProps: IInstalledStatusCellProps) => {
        const { original } = cellProps.row;
        const { value } = cellProps.cell;
        return (
          <InstallStatusCell
            status={value}
            packageToInstall={original.package_available_for_install}
            installedAt={original.last_install?.installed_at}
          />
        );
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
      Cell: (cellProps: ITableNumberCellProps) => (
        <DropdownCell
          placeholder="Actions"
          options={generateActions(
            cellProps.row.original.id,
            cellProps.row.original.status,
            installingSoftwareId,
            canInstall,
            cellProps.row.original.package_available_for_install
          )}
          onChange={(action) => onSelectAction(cellProps.row.original, action)}
        />
      ),
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders };
