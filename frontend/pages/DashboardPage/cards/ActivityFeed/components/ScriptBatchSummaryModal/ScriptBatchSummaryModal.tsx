import React, { useState } from "react";
import { InjectedRouter } from "react-router";

import classnames from "classnames";

import { IActivityDetails } from "interfaces/activity";
import { API_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

import paths from "router/paths";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import { dateAgo } from "utilities/date_format";
import TooltipWrapper from "components/TooltipWrapper";
import { useQuery } from "react-query";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import scriptsAPI, {
  IScriptBatchSummaryQueryKey,
  IScriptBatchSummaryResponse,
} from "services/entities/scripts";
import { AxiosError } from "axios";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Button from "components/buttons/Button";

import ScriptBatchStatusTable from "../ScriptBatchStatusTable";

const baseClass = "script-batch-summary-modal";

interface IScriptBatchSummaryModal {
  scriptBatchExecutionDetails: IActivityDetails;
  onCancel: () => void;
  router: InjectedRouter;
}

const ScriptBatchSummaryModal = ({
  scriptBatchExecutionDetails: details,
  onCancel,
  router,
}: IScriptBatchSummaryModal) => {
  const [showCancelModal, setShowCancelModal] = useState(false);

  const { data: statusData, isLoading, isError } = useQuery<
    IScriptBatchSummaryResponse,
    AxiosError,
    IScriptBatchSummaryResponse,
    IScriptBatchSummaryQueryKey[]
  >(
    [
      {
        scope: "script_batch_summary",
        batch_execution_id: details.batch_execution_id || "",
      },
    ],
    ({ queryKey: [{ batch_execution_id }] }) =>
      scriptsAPI.getRunScriptBatchSummary({ batch_execution_id }),
    {
      enabled: details.batch_execution_id !== undefined,
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const toggleCancelModal = () => {
    setShowCancelModal(!showCancelModal);
  };
  const renderTable = () => {
    if (!details.batch_execution_id || isLoading || !statusData) {
      return <Spinner />;
    }
    if (isError) {
      return <DataError />;
    }
    return (
      <ScriptBatchStatusTable
        statusData={statusData}
        onClickCancel={toggleCancelModal}
      />
    );
  };

  let activityCreatedAt: Date | null = null;
  try {
    activityCreatedAt = new Date(details?.created_at || "");
  } catch (e) {
    // invalid date string
    activityCreatedAt = null;
  }

  const targetedTitle = (
    <TooltipWrapper
      tipContent="The number of hosts originally targeted,
including those where scripts were 
incompatible or cancelled."
    >
      Targeted
    </TooltipWrapper>
  );

  const renderCancelModal = () => {
    const cancelBaseClass = "script-batch-cancel-modal";
    if (!statusData) {
      // the conditions for triggering the cancel modal mean this will never be the case. This is
      // for the TS compiler
      return null;
    }

    let scriptsPath = paths.CONTROLS_SCRIPTS;
    if (statusData.team_id === API_NO_TEAM_ID) {
      // as of this writing these are the same value, 0, but this explicitly delineates API-speak
      // from UI-logic-speak
      scriptsPath += `/?team_id=${APP_CONTEXT_NO_TEAM_ID}`;
    } else if (statusData.team_id) {
      scriptsPath += `/?team_id=${statusData.team_id}`;
    }
    // statusData.team_id == 0 would represent all teams, and is not supported
    return (
      <Modal
        title="Cancel script"
        onExit={toggleCancelModal}
        onEnter={toggleCancelModal}
        className={cancelBaseClass}
      >
        <>
          <div className={`${cancelBaseClass}__content`}>
            <p>
              To cancel all pending runs of this script, edit or delete the
              script.
            </p>
            <div className="modal-cta-wrap">
              <Button onClick={toggleCancelModal}>Done</Button>
              <Button
                onClick={() => router.push(scriptsPath)}
                variant="inverse"
              >
                Go to scripts
              </Button>
            </div>
          </div>
        </>
      </Modal>
    );
  };

  const parentModalClasses = classnames(baseClass, {
    [`${baseClass}__hide-main`]: showCancelModal,
  });

  return (
    <>
      <Modal
        // script_name will always be present at this point
        title={details?.script_name || "Script Batch Summary"}
        onExit={onCancel}
        onEnter={onCancel}
        className={parentModalClasses}
      >
        <div className={`${baseClass}__modal-content`}>
          <div className="header">
            {activityCreatedAt && (
              <DataSet title="Ran" value={dateAgo(activityCreatedAt)} />
            )}
            <DataSet title={targetedTitle} value={details.host_count} />
          </div>
          {renderTable()}
          <div className="modal-cta-wrap">
            <Button onClick={onCancel}>Done</Button>
          </div>
        </div>
      </Modal>
      {showCancelModal && renderCancelModal()}
    </>
  );
};

export default ScriptBatchSummaryModal;
