import React, { useEffect, useRef } from "react";
import ReactTooltip from "react-tooltip";

import {
  IAppLastInstall,
  IDeviceSoftware,
  IDeviceSoftwareWithUiStatus,
  IHostSoftware,
  IHostSoftwareWithUiStatus,
  ISoftwareLastInstall,
  SoftwareInstallStatus,
} from "interfaces/software";
import { dateAgo } from "utilities/date_format";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import { HostInstallerActionButton } from "../../../HostSoftwareLibrary/HostInstallerActionCell/HostInstallerActionCell";
import {
  InstallOrCommandUuid,
  IStatusDisplayConfig,
  RECENT_SUCCESS_ACTION_MESSAGE,
} from "../../InstallStatusCell/InstallStatusCell";

const baseClass = "update-software-item";

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
    tooltip: ({ lastInstalledAt }) =>
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

interface IInstallerInfoProps {
  software: IDeviceSoftware;
}

const InstallerInfo = ({ software }: IInstallerInfoProps) => {
  const {
    name,
    source,
    software_package: installerPackage,
    app_store_app: vppApp,
  } = software;
  return (
    <div className={`${baseClass}__item-topline`}>
      <div className={`${baseClass}__item-icon`}>
        <SoftwareIcon
          url={vppApp?.icon_url}
          name={name}
          source={source}
          size="large"
        />
      </div>
      <div className={`${baseClass}__item-name-version`}>
        <div className={`${baseClass}__item-name`}>
          <TooltipTruncatedText value={name || installerPackage?.name} />
        </div>
        <div className={`${baseClass}__item-version`}>
          {installerPackage?.version || vppApp?.version || ""}
        </div>
      </div>
    </div>
  );
};

type IInstallerStatusProps = Pick<IHostSoftware, "id" | "status"> & {
  last_install: ISoftwareLastInstall | IAppLastInstall | null;
  onShowInstallerDetails: (uuid?: InstallOrCommandUuid) => void;
};

const InstallerStatus = ({
  id,
  status,
  last_install,
  onShowInstallerDetails,
}: IInstallerStatusProps) => {
  const displayConfig = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG];
  if (!displayConfig) {
    // API should ensure this never happens, but just in case
    return null;
  }

  return (
    <div className={`${baseClass}__status-content`}>
      <div
        className={`${baseClass}__status-with-tooltip`}
        data-tip
        data-for={`install-tooltip__${id}`}
      >
        {displayConfig.iconName === "pending-outline" ? (
          <Spinner size="x-small" includeContainer={false} centered={false} />
        ) : (
          <Icon name={displayConfig.iconName || "install"} />
        )}
        {last_install && displayConfig.displayText === "Failed" && (
          <span data-testid={`${baseClass}__status--test`}>
            <Button
              className={`${baseClass}__item-status-button`}
              variant="text-icon"
              onClick={() => {
                onShowInstallerDetails();
              }}
            >
              {displayConfig.displayText}
            </Button>
          </span>
        )}
      </div>
      <ReactTooltip
        className={`${baseClass}__status-tooltip`}
        effect="solid"
        backgroundColor="#3e4771"
        id={`install-tooltip__${id}`}
        data-html
      >
        <span className={`${baseClass}__status-tooltip-text`}>
          {displayConfig.tooltip({
            lastInstalledAt: last_install?.installed_at,
          })}
        </span>
      </ReactTooltip>
    </div>
  );
};

interface IInstallerStatusActionProps {
  software: IHostSoftwareWithUiStatus;
  onInstall: () => void;
  onShowInstallerDetails: (uuid?: InstallOrCommandUuid) => void;
}

const InstallerStatusAction = ({
  software: { id, status, software_package, app_store_app, ui_status },
  onInstall,
  onShowInstallerDetails,
}: IInstallerStatusActionProps) => {
  // TODO: update this if/when we support self-service app store apps
  const lastInstall =
    software_package?.last_install ?? app_store_app?.last_install ?? null;
  // localStatus is used to track the status of the any user-initiated install action

  const isMountedRef = useRef(false);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const showFailedInstallStatus = status === "failed_install";

  const renderPrimaryStatusAction = () => {
    if (ui_status === "updating") {
      return (
        <>
          <Spinner size="x-small" includeContainer={false} centered={false} />{" "}
          Updating...{" "}
        </>
      );
    }
    if (ui_status === "recently_updated") {
      return (
        <>
          <Icon name="success" />
          <TooltipWrapper
            tipContent={RECENT_SUCCESS_ACTION_MESSAGE("updated")}
            showArrow
            underline={false}
            position="top"
          >
            Updated
          </TooltipWrapper>
        </>
      );
    }
    return (
      <HostInstallerActionButton
        baseClass={baseClass}
        disabled={false}
        onClick={onInstall}
        text="Update"
        icon="refresh"
        testId={`${baseClass}__install-button--test`}
      />
    );
  };

  return (
    <div className={`${baseClass}__item-action-status`}>
      <div className={`${baseClass}__item-action`}>
        {renderPrimaryStatusAction()}
      </div>
      {showFailedInstallStatus && (
        <div className={`${baseClass}__item-status`}>
          <InstallerStatus
            id={id}
            status={status}
            last_install={lastInstall}
            onShowInstallerDetails={onShowInstallerDetails}
          />
        </div>
      )}
    </div>
  );
};

interface IUpdateSoftwareItemProps {
  software: IDeviceSoftwareWithUiStatus;
  onClickUpdateAction: (id: number) => void;
  onShowInstallerDetails: (uuid?: InstallOrCommandUuid) => void;
}

const UpdateSoftwareItem = ({
  software,
  onClickUpdateAction,
  onShowInstallerDetails,
}: IUpdateSoftwareItemProps) => {
  return (
    <Card
      borderRadiusSize="large"
      paddingSize="medium"
      className={`${baseClass}__item`}
    >
      <div className={`${baseClass}__item-content`}>
        <InstallerInfo software={software} />
        <InstallerStatusAction
          software={software}
          onInstall={() => onClickUpdateAction(software.id)}
          onShowInstallerDetails={onShowInstallerDetails}
        />
      </div>
    </Card>
  );
};

export default UpdateSoftwareItem;
