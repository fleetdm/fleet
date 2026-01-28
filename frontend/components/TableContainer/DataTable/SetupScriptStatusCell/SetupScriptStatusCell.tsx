import React from "react";

import { SetupStepStatus } from "interfaces/setup";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import Spinner from "components/Spinner";

const baseClass = "setup-script-status-cell";

interface ISetupScriptStatusCell {
  status: SetupStepStatus;
}

const serverToUiStatus = (
  status: SetupStepStatus
): { label: string; icon: IconNames | "spinner" } => {
  switch (status) {
    case "pending":
      return { label: "Pending", icon: "pending-outline" };
    case "running":
      return { label: "Running", icon: "spinner" };
    case "success":
      return { label: "Ran", icon: "success" };
    case "failure":
    case "cancelled":
      return { label: "Failed", icon: "error" };
    default:
      return { label: "Pending", icon: "pending-outline" };
  }
};

const SetupScriptStatusCell = ({ status }: ISetupScriptStatusCell) => {
  const { label, icon } = serverToUiStatus(status);
  return (
    <div className={baseClass}>
      {icon === "spinner" ? <Spinner size="x-small" /> : <Icon name={icon} />}
      {label}
    </div>
  );
};

export default SetupScriptStatusCell;
