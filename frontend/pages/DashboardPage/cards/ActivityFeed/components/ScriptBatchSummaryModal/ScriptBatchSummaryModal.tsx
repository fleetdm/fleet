import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import classnames from "classnames";

import { IActivityDetails } from "interfaces/activity";

import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import { dateAgo } from "utilities/date_format";
import TooltipWrapper from "components/TooltipWrapper";
import { useQuery } from "react-query";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import scriptsAPI, {
  IScriptBatchSummaryQueryKey,
  IScriptBatchSummaryV1,
} from "services/entities/scripts";
import { AxiosError } from "axios";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Button from "components/buttons/Button";

import ScriptBatchStatusTable from "../ScriptBatchStatusTable";

const baseClass = "script-batch-summary-modal";

export type IScriptBatchDetailsForSummary = Pick<
  IActivityDetails,
  "batch_execution_id" | "created_at" | "script_name" | "host_count"
>;

interface IScriptBatchSummaryModal {
  scriptBatchExecutionDetails: IScriptBatchDetailsForSummary;
  onCancel: () => void;
  router: InjectedRouter;
}

const ScriptBatchSummaryModal = ({
  scriptBatchExecutionDetails: details,
  onCancel,
  router,
}: IScriptBatchSummaryModal) => {
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [isCanceling, setIsCanceling] = useState(false);

  const { data: statusData, isLoading, isError } = useQuery<
    IScriptBatchSummaryV1,
    AxiosError,
    IScriptBatchSummaryV1,
    IScriptBatchSummaryQueryKey[]
  >(
    [
      {
        scope: "script_batch_summary",
        batch_execution_id: details.batch_execution_id || "",
      },
    ],
    ({ queryKey: [{ batch_execution_id }] }) =>
      scriptsAPI.getRunScriptBatchSummaryV1({ batch_execution_id }),
    {
      enabled: details.batch_execution_id !== undefined,
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const toggleCancelModal = () => {
    setShowCancelModal(!showCancelModal);
  };

  const renderTable = () => {
    if (
      !details.batch_execution_id ||
      isLoading ||
      !statusData ||
      isCanceling
    ) {
      return <Spinner />;
    }
    if (isError) {
      return <DataError />;
    }
    return (
      <ScriptBatchStatusTable
        statusData={statusData}
        batchExecutionId={details.batch_execution_id || ""}
        onClickCancel={toggleCancelModal}
      />
    );
  };

  let activityCreatedAt: Date | null = null;
  if (details?.created_at) {
    try {
      activityCreatedAt = new Date(details?.created_at || "");
    } catch (e) {
      // invalid date string
      activityCreatedAt = null;
    }
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

  const { renderFlash } = useContext(NotificationContext);

  const onConfirmCancel = useCallback(
    async (batchExecutionId: string) => {
      setIsCanceling(true);

      try {
        await scriptsAPI.cancelScriptBatch(batchExecutionId);
        renderFlash(
          "success",
          <span className={`${baseClass}__success-message`}>
            <span>Successfully canceled script.</span>
          </span>
        );
        setShowCancelModal(false);
        onCancel();
      } catch (error) {
        renderFlash("error", "Could not cancel script. Please try again.");
      } finally {
        setIsCanceling(false);
      }
    },
    [renderFlash]
  );

  const renderCancelModal = () => {
    const cancelBaseClass = "script-batch-cancel-modal";
    if (!statusData) {
      // the conditions for triggering the cancel modal mean this will never be the case. This is
      // for the TS compiler
      return null;
    }

    return (
      <Modal
        title="Cancel script?"
        onExit={toggleCancelModal}
        onEnter={toggleCancelModal}
        className={cancelBaseClass}
      >
        <>
          <div className={`${cancelBaseClass}__content`}>
            <p>
              This will cancel any pending script runs for{" "}
              <b>{details?.script_name || "this script"}</b>.
            </p>
            <p>
              If this script is currently running on a host, it will complete,
              but results won&rsquo;t appear in Fleet.
            </p>
            <p>You cannot undo this action.</p>
            <div className="modal-cta-wrap">
              <Button
                isLoading={isCanceling}
                disabled={isCanceling}
                onClick={() =>
                  onConfirmCancel(details.batch_execution_id || "")
                }
                variant="alert"
              >
                Cancel script
              </Button>
              <Button variant="inverse-alert" onClick={toggleCancelModal}>
                Back
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
