import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import scriptsAPI, {
  IScriptBatchSummaryQueryKey,
  IScriptBatchSummaryV2,
} from "services/entities/scripts";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import paths from "router/paths";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import ActionButtons from "components/buttons/ActionButtons/ActionButtons";

import getWhen from "../helpers";
import CancelScriptBatchModal from "../components/CancelScriptBatchModal";
import { NotificationContext } from "context/notification";

const baseClass = "script-batch-details-page";

interface IScriptBatchDetailsRouteParams {
  batch_execution_id: string;
}

type IScriptBatchDetailsProps = RouteComponentProps<
  undefined,
  IScriptBatchDetailsRouteParams
>;

const ScriptBatchDetailsPage = ({
  router,
  routeParams,
  location,
}: IScriptBatchDetailsProps) => {
  const { batch_execution_id } = routeParams;

  const [showCancelModal, setShowCancelModal] = useState(false);
  const [isCanceling, setIsCanceling] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const { data: batchDetails, isLoading, isError, refetch } = useQuery<
    IScriptBatchSummaryV2,
    AxiosError,
    IScriptBatchSummaryV2,
    IScriptBatchSummaryQueryKey[]
  >(
    [{ scope: "script_batch_summary", batch_execution_id }],
    ({ queryKey }) => scriptsAPI.getRunScriptBatchSummaryV2(queryKey[0]),
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: !!batch_execution_id }
  );

  const onCancelBatch = useCallback(
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
        // TODO - navigate back to script progress page at status that this script was under
        router.push(
          paths.CONTROLS_SCRIPTS_BATCH_PROGRESS +
            (batchDetails?.status ? `?status=${batchDetails?.status}` : "")
        );
      } catch (error) {
        renderFlash("error", "Could not cancel script. Please try again.");
      } finally {
        setIsCanceling(false);
      }
    },
    [batchDetails?.status, renderFlash, router]
  );

  const renderContent = () => {
    if (isLoading || !batchDetails) {
      return <Spinner />;
    }
    const {
      script_name,
      status,

      targeted_host_count: targeted,
      ran_host_count: ran,
      errored_host_count: errored,
      pending_host_count: pending,
      incompatible_host_count: incompatible,
      canceled_host_count: canceled,
    } = batchDetails || {};

    const subTitle = (
      <>
        <span>
          <b>{targeted}</b> hosts targeted (
          {Math.ceil((ran + errored) / targeted)}% responded)
        </span>
        <span className="when">{getWhen(batchDetails)}</span>
      </>
    );

    return (
      <div className={`${baseClass}`}>
        {/* TODO - may need to preserve team, selected batch run state here */}
        <BackLink
          text="Back to script activity"
          path={paths.CONTROLS_SCRIPTS_BATCH_PROGRESS}
        />

        <SectionHeader
          wrapperCustomClass={`${baseClass}__header`}
          title={script_name}
          subTitle={subTitle}
          details={
            <ActionButtons
              baseClass={baseClass}
              actions={[
                {
                  type: "secondary",
                  label: "Show script",
                  buttonVariant: "text-icon",
                  iconName: "eye",
                  onClick: () => {
                    // TODO - implement script viewing logic
                  },
                },
                {
                  type: "secondary",
                  label: "Cancel",
                  onClick: () => {
                    setShowCancelModal(true);
                  },
                  hideAction: status === "finished",
                  buttonVariant: "alert",
                },
              ]}
            />
          }
          alignLeftHeaderVertically
          greySubtitle
        />
        {/* tabs */}
      </div>
    );
  };

  return (
    <>
      {showCancelModal && (
        <CancelScriptBatchModal
          onSubmit={onCancelBatch}
          onExit={() => setShowCancelModal(false)}
          scriptName={batchDetails?.script_name}
          isCanceling={isCanceling}
        />
      )}
      <MainContent className={baseClass}>{renderContent()}</MainContent>
    </>
  );
};

export default ScriptBatchDetailsPage;
