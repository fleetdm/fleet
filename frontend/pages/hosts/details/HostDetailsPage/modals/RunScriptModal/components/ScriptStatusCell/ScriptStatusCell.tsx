import React from "react";
import { formatDistanceToNow } from "date-fns";

import { ILastExecution, IScriptExecutionStatus } from "interfaces/script";

import StatusIndicatorWithIcon, {
  IndicatorStatus,
} from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import TextCell from "components/TableContainer/DataTable/TextCell";

interface IScriptStatusDisplayConfig {
  displayText: string;
  iconStatus: IndicatorStatus;
  tooltip: (executedAt?: string) => string;
}

const STATUS_DISPLAY_CONFIG: Record<
  IScriptExecutionStatus,
  IScriptStatusDisplayConfig
> = {
  ran: {
    displayText: "Ran",
    iconStatus: "success",
    tooltip: (executedAt) =>
      `Script ran and exited with exit code 0 (${executedAt} ago).`,
  },
  pending: {
    displayText: "Pending",
    iconStatus: "pendingPartial",
    tooltip: () => "Script is running or will run when the host comes online.",
  },
  error: {
    displayText: "Error",
    iconStatus: "error",
    tooltip: (executedAt) =>
      `Script ran and exited with a non-zero exit code (${executedAt} ago).`,
  },
};

interface IScriptStatusCellProps {
  lastExecution: ILastExecution | null;
}

const ScriptStatusCell = ({ lastExecution }: IScriptStatusCellProps) => {
  if (!lastExecution) {
    return <TextCell value={null} />;
  }

  const { displayText, iconStatus, tooltip } = STATUS_DISPLAY_CONFIG[
    lastExecution.status
  ];

  const humanizedExecutedAt = formatDistanceToNow(
    new Date(lastExecution.executed_at),
    { includeSeconds: true }
  );

  return (
    <StatusIndicatorWithIcon
      value={displayText}
      status={iconStatus}
      tooltip={{
        tooltipText: tooltip(humanizedExecutedAt),
      }}
    />
  );
};

export default ScriptStatusCell;
