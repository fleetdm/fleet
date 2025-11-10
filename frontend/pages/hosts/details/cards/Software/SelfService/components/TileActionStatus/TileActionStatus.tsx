import React from "react";
import {
  IDeviceSoftwareWithUiStatus,
  IHostSoftwareUiStatus,
} from "interfaces/software";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Spinner from "components/Spinner";

const baseClass = "tile-action-status";

interface TileActionStatusProps {
  software: IDeviceSoftwareWithUiStatus;
  onActionClick: (software: IDeviceSoftwareWithUiStatus) => void;
}

const getTileActionLabel = (uiStatus: IHostSoftwareUiStatus): string | null => {
  switch (uiStatus) {
    case "uninstalled":
    case "recently_uninstalled":
      return "Install";
    case "failed_install":
    case "failed_install_update_available":
    case "failed_script":
      return "Retry";
    case "update_available":
    case "failed_uninstall_update_available":
      return "Update";
    case "installed":
    case "recently_installed":
    case "recently_updated":
    case "failed_uninstall": // Mobile UI only shows install action despite ui_status relating to uninstall
      return "Reinstall";
    case "never_ran_script":
      return "Run";
    case "ran_script":
      return "Rerun";
    default:
      return "Install";
  }
};

const getPendingOrRunningLabel = (
  uiStatus: IHostSoftwareUiStatus
): string | null => {
  switch (uiStatus) {
    case "updating":
    case "pending_update":
      return "Updating...";
    case "installing":
    case "pending_install":
      return "Installing...";
    case "running_script":
    case "pending_script":
      return "Running...";
    case "uninstalling":
    case "pending_uninstall":
      return "Uninstalling...";
    default:
      return null;
  }
};

const TileActionStatus = ({
  software,
  onActionClick,
}: TileActionStatusProps) => {
  const actionLabel = getTileActionLabel(software.ui_status);
  const isError =
    software.ui_status === "failed_install" ||
    software.ui_status === "failed_install_update_available";

  const isActiveAction =
    software.ui_status === "updating" || software.ui_status === "installing";

  const renderActiveActionStatus = () => {
    return (
      <>
        <Spinner size="x-small" includeContainer={false} centered={false} />
        {getPendingOrRunningLabel(software.ui_status)}
      </>
    );
  };

  const renderActionStatus = () => {
    return (
      <>
        {isError && (
          <div className="self-service-tile__item-error">
            <Icon name="error" />
            <div className="self-service-tile__item-error-text">Failed</div>
          </div>
        )}
        {actionLabel && (
          <Button variant="inverse" onClick={() => onActionClick(software)}>
            {actionLabel}
          </Button>
        )}
      </>
    );
  };

  return (
    <div className={baseClass}>
      {isActiveAction ? renderActiveActionStatus() : renderActionStatus()}
    </div>
  );
};

export default TileActionStatus;
