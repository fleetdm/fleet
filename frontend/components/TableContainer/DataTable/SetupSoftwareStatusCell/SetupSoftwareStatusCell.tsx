import React from "react";

import { SetupStepStatus } from "interfaces/software";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import Spinner from "components/Spinner";

const baseClass = "setup-software-status-cell";

interface ISetupSoftwareStatusCell {
  status: SetupStepStatus;
}

const serverToUiStatus = (
  status: SetupStepStatus
): { label: string; icon: IconNames | "spinner" } => {
  switch (status) {
    case "pending":
      return { label: "Pending", icon: "pending-outline" };
    case "running":
      return { label: "Installing", icon: "spinner" };
    case "success":
      return { label: "Installed", icon: "success" };
    case "failure":
    case "cancelled":
      return { label: "Failed", icon: "error" };
    default:
      return { label: "Pending", icon: "pending-outline" };
  }
};

const SetupSoftwareStatusCell = ({ status }: ISetupSoftwareStatusCell) => {
  const { label, icon } = serverToUiStatus(status);
  return (
    <div className={baseClass}>
      {icon === "spinner" ? <Spinner size="x-small" /> : <Icon name={icon} />}
      {label}
    </div>
  );
};

export default SetupSoftwareStatusCell;
