import React, { useState, useRef, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";

import {
  IHostAppStoreApp,
  IHostSoftware,
  IHostSoftwarePackage,
  SoftwareInstallStatus,
  SoftwareSource,
  formatSoftwareType,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
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
import { getDropdownOptionTooltipContent } from "../../HostDetailsPage/HostActionsDropdown/helpers";

export const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "showDetails", label: "Show details", disabled: false },
  { value: "install", label: "Install", disabled: false },
  { value: "uninstall", label: "Uninstall", disabled: false },
];

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

export interface generateActionsProps {
  hostScriptsEnabled: boolean;
  // hostCanWriteSoftware: boolean; // TODO: this is not used in the component, but it is passed as a prop why
  softwareId: number;
  status: SoftwareInstallStatus | null;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  hostMDMEnrolled?: boolean;
}

interface IHostSWLibraryTableHeaders {
  userHasSWWritePermission: boolean;
  hostScriptsEnabled?: boolean;
  // hostCanWriteSoftware: boolean;
  router: InjectedRouter;
  teamId: number;
  hostMDMEnrolled?: boolean;
  baseClass: string;
  onShowSoftwareDetails?: (software?: IHostSoftware) => void;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (softwareId: number) => void;
  isHostOnline: boolean;
}

interface IInstallerStatusActionsProps {
  software: IHostSoftware;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (softwareId: number) => void;
  baseClass: string;
}

interface IButtonActionState {
  installDisabled: boolean;
  installTooltip?: string | JSX.Element;
  uninstallDisabled: boolean;
  uninstallTooltip?: string | JSX.Element;
}

const getButtonActionState = ({
  hostScriptsEnabled,
  // softwareIdActionPending,
  softwareId,
  status,
  app_store_app,
  hostMDMEnrolled,
}: generateActionsProps): IButtonActionState => {
  const pendingStatuses = ["pending_install", "pending_uninstall"];
  let installDisabled = false;
  let uninstallDisabled = false;
  let installTooltip: JSX.Element | string | undefined;
  let uninstallTooltip: JSX.Element | string | undefined;

  if (!hostScriptsEnabled && !app_store_app) {
    installDisabled = true;
    uninstallDisabled = true;
    installTooltip = getDropdownOptionTooltipContent("installSoftware");
    uninstallTooltip = getDropdownOptionTooltipContent("uninstallSoftware");
  }

  if (pendingStatuses.includes(status || "")) {
    installDisabled = true;
    uninstallDisabled = true;
  }

  if (app_store_app) {
    // Hidden uninstall button for app store apps but disabled just in case
    uninstallDisabled = true;

    if (!hostMDMEnrolled) {
      installDisabled = true;
      installTooltip = "To install, turn on MDM for this host.";
    }
  }

  return {
    installDisabled,
    installTooltip: installTooltip || undefined,
    uninstallDisabled,
    uninstallTooltip: uninstallTooltip || undefined,
  };
};

const InstallerStatusAction = ({
  software,
  onClickInstallAction,
  onClickUninstallAction,
  baseClass,
  hostScriptsEnabled,
  hostMDMEnrolled,
}: IInstallerStatusActionsProps & Partial<generateActionsProps>) => {
  const { id, status, software_package, app_store_app } = software;

  const {
    installDisabled,
    installTooltip,
    uninstallDisabled,
    uninstallTooltip,
  } = getButtonActionState({
    hostScriptsEnabled: hostScriptsEnabled || false,
    softwareId: id,
    status,
    app_store_app,
    hostMDMEnrolled,
    software_package,
  });

  // const { renderFlash } = useContext(NotificationContext);

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

  const isMountedRef = useRef(false);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  return (
    <div className={`${baseClass}__item-actions`}>
      <div className={`${baseClass}__item-action`}>
        <TooltipWrapper
          tipContent={installTooltip}
          underline={false}
          showArrow
          position="top"
        >
          <Button
            variant="text-icon"
            type="button"
            className={`${baseClass}__item-action-button`}
            onClick={() => onClickInstallAction(id)}
            disabled={installDisabled}
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
        </TooltipWrapper>
      </div>
      <div className={`${baseClass}__item-action`}>
        {app_store_app
          ? null
          : software_package && (
              <TooltipWrapper
                tipContent={uninstallTooltip}
                underline={false}
                showArrow
                position="top"
              >
                <Button
                  variant="text-icon"
                  type="button"
                  className={`${baseClass}__item-action-button`}
                  onClick={() => onClickUninstallAction(id)}
                  disabled={uninstallDisabled}
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
              </TooltipWrapper>
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
  // hostCanWriteSoftware,
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

  // we want to hide the install/uninstall actions if the user doesn't have write permission
  if (!userHasSWWritePermission) {
    tableHeaders.pop();
  }
  return tableHeaders;
};

export default {
  generateHostSWLibraryTableHeaders,
};
