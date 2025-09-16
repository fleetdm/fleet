import React from "react";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "script-batch-host-count-cell";

interface IScriptBatchHostCountCellProps {
  batchExecutionId: string;
  status: string;
  count: number;
  onClickCancel: () => void;
  teamId?: number;
}

const ScriptBatchHostCountCell = ({
  batchExecutionId,
  status,
  count,
  onClickCancel,
  teamId,
}: IScriptBatchHostCountCellProps) => {
  const hostPath = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams({
    script_batch_execution_status: status,
    script_batch_execution_id: batchExecutionId,
    team_id: teamId,
  })}`;

  const renderCancelButton = () => {
    if (status !== "pending" || count === 0) {
      return null;
    }

    return (
      <Button
        className={`${baseClass}__cancel-button`}
        onClick={onClickCancel}
        variant="inverse"
      >
        Cancel
      </Button>
    );
  };

  return (
    <div className={baseClass}>
      <CustomLink url={hostPath} text={count.toString()} />
      {renderCancelButton()}
    </div>
  );
};

export default ScriptBatchHostCountCell;
