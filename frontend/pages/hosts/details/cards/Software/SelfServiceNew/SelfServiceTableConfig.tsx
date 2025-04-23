import React, { useRef, useEffect, useCallback, useContext } from "react";
import { CellProps, Column } from "react-table";
import ReactTooltip from "react-tooltip";

import {
  IAppLastInstall,
  IHostSoftware,
  ISoftwareLastInstall,
  SoftwareInstallStatus,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import deviceApi from "services/entities/device_user";
import { NotificationContext } from "context/notification";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { dateAgo } from "utilities/date_format";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import Spinner from "components/Spinner";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { IStatusDisplayConfig } from "../InstallStatusCell/InstallStatusCell";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IVulnerabilitiesCellProps = IInstalledVersionsCellProps;

const baseClass = "self-service-table";

const STATUS_CONFIG: Record<
  Exclude<
    SoftwareInstallStatus,
    "pending_uninstall" | "failed_uninstall" | "uninstalled"
  >,
  IStatusDisplayConfig
> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ lastInstalledAt = "" }) =>
      `Software is installed (${dateAgo(lastInstalledAt as string)}).`,
  },
  pending_install: {
    iconName: "pending-outline",
    displayText: "Installing...",
    tooltip: () => "Fleet is installing software.",
  },
  failed_install: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ lastInstalledAt = "" }) => (
      <>
        Software failed to install
        {lastInstalledAt ? ` (${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
        <b>Retry</b> to install again, or contact your IT department.
      </>
    ),
  },
};

type IInstallerStatusProps = Pick<IHostSoftware, "id" | "status"> & {
  last_install: ISoftwareLastInstall | IAppLastInstall | null;
  onShowInstallerDetails: (installId: string) => void;
};

const InstallerStatus = ({
  id,
  status,
  last_install,
  onShowInstallerDetails,
}: IInstallerStatusProps) => {
  const displayConfig = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG];
  if (!displayConfig) {
    // This is shown mid-install
    return <>{DEFAULT_EMPTY_CELL_VALUE}</>;
  }

  return (
    <TooltipWrapper
      tipContent={displayConfig.tooltip({
        lastInstalledAt: last_install?.installed_at,
      })}
      showArrow
      underline={false}
      position="top"
    >
      <div className={`${baseClass}__status-content`}>
        {displayConfig.iconName === "pending-outline" ? (
          <Spinner size="x-small" includeContainer={false} centered={false} />
        ) : (
          <Icon name={displayConfig.iconName} />
        )}
        <span data-testid={`${baseClass}__status--test`}>
          {last_install && displayConfig.displayText === "Failed" ? (
            <Button
              className={`${baseClass}__item-status-button`}
              variant="text-icon"
              onClick={() => {
                if ("command_uuid" in last_install) {
                  onShowInstallerDetails(last_install.command_uuid);
                } else if ("install_uuid" in last_install) {
                  onShowInstallerDetails(last_install.install_uuid);
                } else {
                  onShowInstallerDetails("");
                }
                console.log("opened");
              }}
            >
              {displayConfig.displayText}
            </Button>
          ) : (
            displayConfig.displayText
          )}
        </span>
      </div>
    </TooltipWrapper>
  );
};

interface IInstallerStatusActionProps {
  deviceToken: string;
  software: IHostSoftware;
  onInstall: () => void;
  onShowInstallerDetails: (installId: string) => void;
}

const getInstallButtonText = (status: SoftwareInstallStatus | null) => {
  switch (status) {
    case null:
      return "Install";
    case "failed_install":
      return "Retry";
    case "installed":
      return "Reinstall";
    default:
      return "";
  }
};

const getInstallButtonIcon = (status: SoftwareInstallStatus | null) => {
  switch (status) {
    case null:
      return "install";
    case "failed_install":
      return "refresh";
    case "installed":
      return "refresh";
    default:
      return undefined;
  }
};

const InstallerStatusAction = ({
  deviceToken,
  software: { id, status, software_package, app_store_app },
  onInstall,
  onShowInstallerDetails,
}: IInstallerStatusActionProps) => {
  const { renderFlash } = useContext(NotificationContext);

  // TODO: update this if/when we support self-service app store apps
  const last_install =
    software_package?.last_install ?? app_store_app?.last_install ?? null;

  // localStatus is used to track the status of the any user-initiated install action
  const [localStatus, setLocalStatus] = React.useState<
    SoftwareInstallStatus | undefined
  >(undefined);

  const installButtonText = getInstallButtonText(status);
  const installButtonIcon = getInstallButtonIcon(status);

  // if the localStatus is "failed", we don't want our tooltip to include the old installed_at date so we
  // set this to null, which tells the tooltip to omit the parenthetical date
  const lastInstall = localStatus === "failed_install" ? null : last_install;

  const isMountedRef = useRef(false);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const onClick = useCallback(async () => {
    setLocalStatus("pending_install");
    try {
      await deviceApi.installSelfServiceSoftware(deviceToken, id);
      if (isMountedRef.current) {
        onInstall();
      }
    } catch (error) {
      renderFlash("error", "Couldn't install. Please try again.");
      if (isMountedRef.current) {
        setLocalStatus("failed_install");
      }
    }
  }, [deviceToken, id, onInstall, renderFlash]);

  return (
    <div className={`${baseClass}__item-status-action`}>
      <div className={`${baseClass}__item-action`}>
        {installButtonText ? (
          <Button
            variant="text-icon"
            type="button"
            className={`${baseClass}__item-action-button`}
            onClick={onClick}
            disabled={localStatus === "pending_install"}
          >
            {installButtonIcon && (
              <Icon
                name={installButtonIcon}
                color="core-fleet-blue"
                size="small"
              />
            )}
            <span data-testid={`${baseClass}__action-button--test`}>
              {installButtonText}
            </span>
          </Button>
        ) : (
          DEFAULT_EMPTY_CELL_VALUE
        )}
      </div>
    </div>
  );
};

export const generateSoftwareTableData = (
  software: IHostSoftware[]
): IHostSoftware[] => {
  return software;
};

interface ISelfServiceTableHeaders {
  deviceToken: string;
  onInstall: () => void;
  onShowInstallerDetails: (installId: string) => void;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  deviceToken,
  onInstall,
  onShowInstallerDetails,
}: ISelfServiceTableHeaders): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      disableGlobalFilter: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, source } = cellProps.row.original;
        return (
          <SoftwareNameCell
            name={name}
            source={source}
            myDevicePage
            isSelfService
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Status"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      disableGlobalFilter: true,
      accessor: "source",
      Cell: (cellProps: ITableStringCellProps) => (
        <InstallerStatus
          id={cellProps.row.original.id}
          status={cellProps.row.original.status}
          last_install={
            cellProps.row.original.software_package?.last_install || null
          }
          onShowInstallerDetails={onShowInstallerDetails}
        />
      ),
    },
    {
      Header: "Actions",
      accessor: (originalRow) => originalRow.status,
      disableSortBy: true,
      Cell: (cellProps: IVulnerabilitiesCellProps) => {
        return (
          <InstallerStatusAction
            deviceToken={deviceToken}
            software={cellProps.row.original}
            onInstall={onInstall}
            onShowInstallerDetails={onShowInstallerDetails}
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders, generateSoftwareTableData };
