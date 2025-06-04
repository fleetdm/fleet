import React, {
  useState,
  useContext,
  useRef,
  useEffect,
  useCallback,
} from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";
import { NotificationContext } from "context/notification";

import {
  IHostAppStoreApp,
  IHostSoftware,
  IHostSoftwarePackage,
  SoftwareInstallStatus,
  SoftwareSource,
  formatSoftwareType,
} from "interfaces/software";
import {
  IHeaderProps,
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import VersionCell from "pages/SoftwarePage/components/tables/VersionCell";
import InstallStatusCell from "../Software/InstallStatusCell";
import {
  getInstallButtonIcon,
  getInstallButtonText,
  getUninstallButtonIcon,
  getUninstallButtonText,
  DisplayActionItems,
} from "../Software/SelfService/SelfServiceTableConfig";

export const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "showDetails", label: "Show details", disabled: false },
  { value: "install", label: "Install", disabled: false },
  { value: "uninstall", label: "Uninstall", disabled: false },
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
type IActionCellProps = CellProps<IHostSoftware, IHostSoftware["status"]>;

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

interface IHostSWLibraryTableHeaderProps {
  userHasSWWritePermission: boolean;
  hostScriptsEnabled?: boolean;
  hostCanWriteSoftware: boolean;
  softwareIdActionPending: number | null;
  router: InjectedRouter;
  teamId: number;
  hostMDMEnrolled?: boolean;
  baseClass: string;
}

interface IInstallerStatusActionsProps {
  software: IHostSoftware;
  onInstallOrUninstall: () => void;
  onClickUninstallAction: (software: IHostSoftware) => void;
  baseClass: string;
}

const InstallerStatusAction = ({
  software: { id, status, software_package, app_store_app },
  onInstallOrUninstall,
  onClickUninstallAction,
  baseClass,
}: IInstallerStatusActionsProps) => {
  const { renderFlash } = useContext(NotificationContext);

  // displayActionItems is used to track the display text and icons of the install and uninstall button
  const [
    displayActionItems,
    setDisplayActionItems,
  ] = useState<DisplayActionItems>({
    install: {
      text: getInstallButtonText(status),
      icon: getInstallButtonIcon(status),
    },
    uninstall: {
      text: getUninstallButtonText(status),
      icon: getUninstallButtonIcon(status),
    },
  });

  useEffect(() => {
    // We update the text/icon only when we see a change to a non-pending status
    // Pending statuses keep the original text shown (e.g. "Retry" text on failed
    // install shouldn't change to "Install" text because it was clicked and went
    // pending. Once the status is no longer pending, like 'installed' the
    // text will update to "Reinstall")
    if (status !== "pending_install" && status !== "pending_uninstall") {
      setDisplayActionItems({
        install: {
          text: getInstallButtonText(status),
          icon: getInstallButtonIcon(status),
        },
        uninstall: {
          text: getUninstallButtonText(status),
          icon: getUninstallButtonIcon(status),
        },
      });
    }
  }, [status]);

  const isAppStoreApp = !!app_store_app;
  const canUninstallSoftware = !isAppStoreApp && !!software_package;

  const isMountedRef = useRef(false);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const onClickInstallAction = useCallback(async () => {
    console.log("TODO");
  }, []);

  return (
    <div className={`${baseClass}__item-actions`}>
      <div className={`${baseClass}__item-action`}>
        <Button
          variant="text-icon"
          type="button"
          className={`${baseClass}__item-action-button`}
          onClick={onClickInstallAction}
          disabled={
            status === "pending_install" || status === "pending_uninstall"
          }
        >
          <Icon
            name={displayActionItems.install.icon}
            color="core-fleet-blue"
            size="small"
          />

          <span data-testid={`${baseClass}__install-button--test`}>
            {displayActionItems.install.text}
          </span>
        </Button>
      </div>
      <div className={`${baseClass}__item-action`}>
        {canUninstallSoftware && (
          <Button
            variant="text-icon"
            type="button"
            className={`${baseClass}__item-action-button`}
            onClick={onClickUninstallAction}
            disabled={
              status === "pending_install" || status === "pending_uninstall"
            }
          >
            <Icon
              name={displayActionItems.uninstall.icon}
              color="core-fleet-blue"
              size="small"
            />
            <span data-testid={`${baseClass}__uninstall-button--test`}>
              {displayActionItems.uninstall.text}
            </span>
          </Button>
        )}
      </div>
    </div>
  );
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateHostSWLibraryTableHeaders = ({
  userHasSWWritePermission,
  hostScriptsEnabled = false,
  hostCanWriteSoftware,
  softwareIdActionPending,
  router,
  teamId,
  hostMDMEnrolled,
  baseClass,
}: IHostSWLibraryTableHeaderProps): ISoftwareTableConfig[] => {
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
      Header: () => (
        <HeaderCell
          disableSortBy
          value={
            <TooltipWrapper
              tipContent={
                <>
                  The status of the last time <br />
                  Fleet attempted an install <br />
                  or uninstall.
                </>
              }
            >
              Install status
            </TooltipWrapper>
          }
        />
      ),
      disableSortBy: true,
      accessor: "status",
      Cell: ({ row: { original } }: IInstalledStatusCellProps) => {
        return <InstallStatusCell {...original} />;
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
      Header: "Installer version",
      id: "installer_version",
      disableSortBy: true,
      // we use function as accessor because we have two columns that
      // need to access the same data. This is not supported with a string
      // accessor.
      accessor: (originalRow) =>
        originalRow.software_package || originalRow.app_store_app,
      Cell: (cellProps: IInstalledVersionsCellProps) => {
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
      Header: "Actions",
      accessor: (originalRow) => originalRow.status,
      disableSortBy: true,
      Cell: (cellProps: IActionCellProps) => {
        return (
          <InstallerStatusAction
            software={cellProps.row.original}
            onInstallOrUninstall={() => {
              console.log("TODO: install or uninstall action");
            }}
            onClickUninstallAction={() => console.log("TODO: uninstall action")}
            baseClass={baseClass}
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default {
  generateHostSWLibraryTableHeaders,
};
