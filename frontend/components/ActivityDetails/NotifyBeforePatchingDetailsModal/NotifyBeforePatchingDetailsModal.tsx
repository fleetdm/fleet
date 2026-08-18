import React from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { IActivityDetails } from "interfaces/activity";
import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import DataSet from "components/DataSet";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import Textarea from "components/Textarea";

const baseClass = "notify-before-patching-details-modal";

// TODO(#50915): Marko will publish a "another notification is displayed" exit
// code — waiting on the number. Meanwhile the "no script_execution_id" branch
// covers the dispatcher-caught case (see spec: absence of execution_id is
// unambiguous because the patch kind sets no `expires_at`).
const DEFERRED_EXIT_CODE_TBC = -1;

const FAILURE_COPY_BY_EXIT_CODE: Record<number, string> = {
  0: "If the host is offline when the patch is forced, Fleet skips the patch. When the host comes back online Fleet notifies the end user again and the patch is forced 1 hour later.",
  30: "The notification couldn't load. Fleet will try again on the next policy run.",
  31: "The notification couldn't load. Fleet will try again on the next policy run.",
  41: "The screen was locked so the end user couldn't see the notification. Fleet will try again on the next policy run.",
  100: "The Fleet Desktop app is required to notify end users. Add the app from the Fleet-maintained catalog and deploy to all your hosts.",
  101: "The Fleet Desktop app v1.5.0 is required to notify end users. Add the app from the Fleet-maintained catalog and deploy to all your hosts.",
  [DEFERRED_EXIT_CODE_TBC]:
    "Another notification was displayed. Fleet will try again on the next policy run.",
};

const DEFERRED_SENTENCE =
  "Another notification was displayed. Fleet will try again on the next policy run.";

const getFailureSentence = (
  scriptExecutionId?: string,
  exitCode?: number | null
): string | null => {
  // A server-side deferral emits an activity with no script run — absence of
  // script_execution_id is the unambiguous signal.
  if (!scriptExecutionId) return DEFERRED_SENTENCE;
  if (exitCode === null || exitCode === undefined) return null;
  return FAILURE_COPY_BY_EXIT_CODE[exitCode] ?? null;
};

interface INotifyBeforePatchingDetailsModalProps {
  details: IActivityDetails;
  onCancel: () => void;
}

const NotifyBeforePatchingDetailsModal = ({
  details,
  onCancel,
}: INotifyBeforePatchingDetailsModalProps) => {
  const {
    host_display_name: hostName,
    software_titles: titles = [],
    status,
    time_before: timeBefore,
    script_execution_id: scriptExecutionId,
  } = details;

  const timeLabel = timeBefore === 300 ? "5 minutes" : "1 hour";
  const failed = status === "failed";

  const { data: scriptResult, isLoading, isError } = useQuery<
    IScriptResultResponse,
    AxiosError
  >(
    ["notify-script-result", scriptExecutionId],
    () => scriptsAPI.getScriptResult(scriptExecutionId as string),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // Only fetch when the server actually ran a script. A dispatcher-caught
      // deferral has no execution id and we render the deferred sentence
      // without hitting the API.
      enabled: !!scriptExecutionId,
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
    }
  );

  const failureSentence = failed
    ? getFailureSentence(scriptExecutionId, scriptResult?.exit_code)
    : null;

  const renderContent = () => {
    if (scriptExecutionId && isLoading) {
      return <Spinner />;
    }
    if (scriptExecutionId && isError) {
      return <DataError description="Close this modal and try again." />;
    }

    const verb = failed ? "failed to notify" : "notified";
    return (
      <div className={`${baseClass}__content`}>
        <p className={`${baseClass}__status`}>
          Fleet {verb} end user {timeLabel} before patching on{" "}
          <b>{hostName || "this host"}</b>.
        </p>
        {failureSentence && (
          <p className={`${baseClass}__failure`}>{failureSentence}</p>
        )}
        <DataSet
          title="Apps"
          value={
            titles.length
              ? titles.map((t) => getDisplayedSoftwareName(t)).join(", ")
              : "—"
          }
        />
        {scriptResult?.output && (
          <Textarea label="Notification script output:" variant="code">
            {scriptResult.output}
          </Textarea>
        )}
      </div>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Notification details"
      onExit={onCancel}
      onEnter={onCancel}
    >
      <>
        {renderContent()}
        <ModalFooter
          primaryButtons={<Button onClick={onCancel}>Done</Button>}
        />
      </>
    </Modal>
  );
};

export default NotifyBeforePatchingDetailsModal;
