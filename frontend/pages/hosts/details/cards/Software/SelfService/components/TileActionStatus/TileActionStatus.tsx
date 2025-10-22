import React from "react";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Spinner from "components/Spinner";

const baseClass = "tile-action-status";

interface TileActionStatusProps {
  software: IDeviceSoftwareWithUiStatus;
  onActionClick: (software: IDeviceSoftwareWithUiStatus) => void;
}

const getTileActionLabel = (software: IDeviceSoftwareWithUiStatus) => {
  if (software.ui_status === "uninstalled") {
    return "Install";
  }
  if (
    software.ui_status === "failed_install" ||
    software.ui_status === "failed_install_update_available"
  ) {
    return "Retry";
  }
  if (software.ui_status === "update_available") {
    return "Update";
  }
  if (software.ui_status === "installed") {
    return "Reinstall";
  }
  return null;
};

const TileActionStatus = ({
  software,
  onActionClick,
}: TileActionStatusProps) => {
  const actionLabel = getTileActionLabel(software);
  const isError =
    software.ui_status === "failed_install" ||
    software.ui_status === "failed_install_update_available";

  const isActiveAction =
    software.ui_status === "updating" || software.ui_status === "installing";

  const renderActiveActionStatus = () => {
    return (
      <>
        <Spinner size="x-small" includeContainer={false} centered={false} />
        {software.ui_status === "updating" ? "Updating..." : "Installing..."}
      </>
    );
  };

  const renderActionStatus = () => {
    return (
      <>
        {isError && (
          <div className="self-service-tile__item-error">
            <Icon name="error" />
            Failed
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
