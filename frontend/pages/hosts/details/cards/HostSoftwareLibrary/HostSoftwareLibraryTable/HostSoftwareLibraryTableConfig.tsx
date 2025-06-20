import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";

import { IHostAppStoreApp, IHostSoftware } from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { getAutomaticInstallPoliciesCount } from "pages/SoftwarePage/helpers";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import VersionCell from "pages/SoftwarePage/components/tables/VersionCell";
import InstallerActionCell from "../InstallerActionCell";
import InstallStatusCell from "../../Software/InstallStatusCell";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledStatusCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["status"]
>;
type IVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IActionCellProps = CellProps<IHostSoftware, IHostSoftware["status"]>;

interface IHostSWLibraryTableHeaders {
  userHasSWWritePermission: boolean;
  hostScriptsEnabled?: boolean;
  router: InjectedRouter;
  teamId: number;
  hostMDMEnrolled?: boolean;
  baseClass: string;
  onShowSoftwareDetails?: (software?: IHostSoftware) => void;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (softwareId: number) => void;
  isHostOnline: boolean;
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
  onClickInstallAction,
  onClickUninstallAction,
  isHostOnline,
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

        const hasPackage = !!app_store_app || !!software_package;
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
            hasPackage={hasPackage}
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
      accessor: "status",
      Cell: ({ row: { original } }: IInstalledStatusCellProps) => {
        return (
          <InstallStatusCell
            software={original}
            onShowSoftwareDetails={onShowSoftwareDetails}
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
      accessor: (originalRow) => originalRow.status,
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
