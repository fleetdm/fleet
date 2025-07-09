import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";

import { IHostAppStoreApp, IHostSoftware } from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { getAutomaticInstallPoliciesCount } from "pages/SoftwarePage/helpers";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import VersionCell from "pages/SoftwarePage/components/tables/VersionCell";
import InstallerActionCell from "../InstallerActionCell";
import InstallStatusCell from "../../Software/InstallStatusCell";

export type IUiStatus =
  | "installed"
  | "uninstalled"
  | "installing"
  | "uninstalling"
  | "updating"
  | "pending_install"
  | "pending_uninstall"
  | "pending_update"
  | "failed_install"
  | "failed_uninstall"
  | "update_available";

export interface IHostSoftwareUi extends IHostSoftware {
  ui_status: IUiStatus;
}

type ISoftwareTableConfig = Column<IHostSoftwareUi>;
type ITableHeaderProps = IHeaderProps<IHostSoftwareUi>;
type ITableStringCellProps = IStringCellProps<IHostSoftwareUi>;
type IInstalledStatusCellProps = CellProps<
  IHostSoftwareUi,
  IHostSoftwareUi["ui_status"]
>;
type IVersionsCellProps = CellProps<
  IHostSoftwareUi,
  IHostSoftwareUi["installed_versions"]
>;
type IActionCellProps = CellProps<IHostSoftwareUi, IHostSoftwareUi["status"]>;

interface IHostSWLibraryTableHeaders {
  userHasSWWritePermission: boolean;
  hostScriptsEnabled?: boolean;
  router: InjectedRouter;
  teamId: number;
  hostMDMEnrolled?: boolean;
  baseClass: string;
  onShowSoftwareDetails?: (software?: IHostSoftware) => void;
  onShowUninstallDetails: (details?: ISoftwareUninstallDetails) => void;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (softwareId: number) => void;
  isHostOnline: boolean;
  hostName: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateHostSWLibraryTableHeaders = ({
  userHasSWWritePermission,
  hostScriptsEnabled = false,
  router,
  teamId,
  hostMDMEnrolled,
  baseClass,
  onShowSoftwareDetails,
  onShowUninstallDetails,
  onClickInstallAction,
  onClickUninstallAction,
  isHostOnline,
  hostName,
}: IHostSWLibraryTableHeaders): ISoftwareTableConfig[] => {
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
          source,
          app_store_app,
          software_package,
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

        return (
          <SoftwareNameCell
            name={name}
            source={source}
            iconUrl={app_store_app?.icon_url}
            path={softwareTitleDetailsPath}
            router={router}
            hasInstaller={hasInstaller}
            isSelfService={isSelfService}
            automaticInstallPoliciesCount={automaticInstallPoliciesCount}
            pageContext="hostDetailsLibrary"
          />
        );
      },
    },
    {
      Header: () => <HeaderCell disableSortBy value="Status" />,
      disableSortBy: true,
      accessor: "ui_status",
      Cell: ({ row: { original } }: IInstalledStatusCellProps) => {
        return (
          <InstallStatusCell
            software={original}
            onShowSoftwareDetails={onShowSoftwareDetails}
            onShowUninstallDetails={onShowUninstallDetails}
            isHostOnline={isHostOnline}
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
      Cell: (cellProps: IVersionsCellProps) => {
        return <VersionCell versions={cellProps.cell.value} />;
      },
    },
    {
      Header: "Library version",
      id: "library_version",
      disableSortBy: true,
      // we use function as accessor because we have two columns that
      // need to access the same data. This is not supported with a string
      // accessor.
      accessor: (originalRow) =>
        originalRow.software_package || originalRow.app_store_app,
      Cell: (cellProps: IVersionsCellProps) => {
        const softwareTitle = cellProps.row.original;
        const installerData = softwareTitle.software_package
          ? softwareTitle.software_package
          : (softwareTitle.app_store_app as IHostAppStoreApp);
        return (
          <VersionCell versions={[{ version: installerData?.version || "" }]} />
        );
      },
    },
    {
      Header: "Actions",
      accessor: (originalRow) => originalRow.ui_status,
      disableSortBy: true,
      Cell: (cellProps: IActionCellProps) => {
        return (
          <InstallerActionCell
            software={cellProps.row.original}
            onClickInstallAction={onClickInstallAction}
            onClickUninstallAction={onClickUninstallAction}
            baseClass={baseClass}
            hostScriptsEnabled={hostScriptsEnabled}
            hostMDMEnrolled={hostMDMEnrolled}
          />
        );
      },
    },
  ];

  // Hide the install/uninstall actions if the user doesn't have write permission
  if (!userHasSWWritePermission) {
    tableHeaders.pop();
  }
  return tableHeaders;
};

export default {
  generateHostSWLibraryTableHeaders,
};
