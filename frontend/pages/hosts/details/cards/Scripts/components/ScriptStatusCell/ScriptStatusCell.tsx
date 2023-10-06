import React from "react";

import { IScriptExecutionStatus } from "services/entities/scripts";

import StatusIndicatorWithIcon, {
  IndicatorStatus,
} from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import TextCell from "components/TableContainer/DataTable/TextCell";

interface IScriptStatusDisplayConfig {
  displayText: string;
  iconStatus: IndicatorStatus;
  tooltip: string;
}

const STATUS_DISPLAY_CONFIG: Record<
  IScriptExecutionStatus,
  IScriptStatusDisplayConfig
> = {
  ran: {
    displayText: "Ran",
    iconStatus: "success",
    tooltip: "Script ran and exited with exit code 0. (5 minutes ago)",
  },
  pending: {
    displayText: "Pending",
    iconStatus: "pendingPartial",
    tooltip: "Script will run when the host comes online.",
  },
  error: {
    displayText: "Error",
    iconStatus: "error",
    tooltip: "Script ran and exited with a non-zero exit code. (5 minutes ago)",
  },
};

interface IScriptStatusCellProps {
  status: IScriptExecutionStatus | null;
}

const ScriptStatusCell = ({ status }: IScriptStatusCellProps) => {
  if (!status) {
    return <TextCell value={status} />;
  }

  const { displayText, iconStatus, tooltip } = STATUS_DISPLAY_CONFIG[status];
  return (
    <StatusIndicatorWithIcon
      value={displayText}
      status={iconStatus}
      tooltip={{
        tooltipText: tooltip,
      }}
    />
  );
};

export default ScriptStatusCell;
