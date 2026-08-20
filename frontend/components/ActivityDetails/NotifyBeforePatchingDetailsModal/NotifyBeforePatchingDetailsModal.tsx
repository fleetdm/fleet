import React, { useState } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { IActivityDetails, INotifyActivityStatus } from "interfaces/activity";
import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import DataError from "components/DataError";
import DataSet from "components/DataSet";
import IconStatusMessage from "components/IconStatusMessage";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import Textarea from "components/Textarea";
import TooltipWrapper from "components/TooltipWrapper";

import CustomLink from "components/CustomLink";

import {
  getCaveatMessage,
  EXIT_CODES_NEEDING_EUE_LINK,
  INLINE_APP_LIMIT,
  PATCHING_END_USER_EXPERIENCE_URL,
  renderNotifyTitleList,
  formatNotifyTimeLabel,
  isNotifyFailure,
} from "./helpers";

const baseClass = "notify-before-patching-details-modal";

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

  const timeLabel = formatNotifyTimeLabel(timeBefore);
  const failed = isNotifyFailure(status as INotifyActivityStatus | undefined);

  const [showDetails, setShowDetails] = useState(false);

  const { data: scriptResult, isLoading, isError } = useQuery<
    IScriptResultResponse,
    AxiosError
  >(
    ["notify-script-result", scriptExecutionId],
    () => scriptsAPI.getScriptResult(scriptExecutionId as string),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // Skip the fetch on dispatcher-caught deferrals (no execution id).
      enabled: !!scriptExecutionId,
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
    }
  );

  const explanation = getCaveatMessage(
    failed,
    scriptExecutionId,
    scriptResult?.exit_code
  );

  const renderContent = () => {
    if (scriptExecutionId && isLoading) {
      return <Spinner />;
    }
    if (scriptExecutionId && isError) {
      return <DataError description="Close this modal and try again." />;
    }

    const verb = failed ? "failed to notify" : "notified";
    // On a silent success (exit 0, no output) the output block would just read
    // "Exit code: 0" — noise, skip it.
    const outputBlock = (() => {
      const code = scriptResult?.exit_code;
      const output = scriptResult?.output;
      if (code == null) return null;
      if (code === 0 && !output) return null;
      return output ? `Exit code: ${code}\n${output}` : `Exit code: ${code}`;
    })();
    // Nothing to reveal for a dispatcher-caught deferral (no script ran) or a
    // silent success (no script contents, no output).
    const hasDetailsContent = !!scriptResult?.script_contents || !!outputBlock;

    const titleList = renderNotifyTitleList(titles);
    // Apps row is redundant when the intro already lists everything.
    const showAppsRow = titles.length > INLINE_APP_LIMIT;

    return (
      <div className={`${baseClass}__content`}>
        <IconStatusMessage
          className={`${baseClass}__status-message`}
          iconName={failed ? "error-outline" : "success-outline"}
          message={
            <span>
              Fleet {verb} end user {timeLabel} before patching
              {titleList && <> {titleList}</>} on{" "}
              <strong>{hostName || "this host"}</strong>.
            </span>
          }
        />
        {explanation && (
          <p className={`${baseClass}__explanation`}>
            {explanation}
            {scriptResult?.exit_code != null &&
              EXIT_CODES_NEEDING_EUE_LINK.has(scriptResult.exit_code) && (
                <>
                  {" "}
                  <CustomLink
                    url={PATCHING_END_USER_EXPERIENCE_URL}
                    text="End user experience"
                    newTab
                  />
                </>
              )}
          </p>
        )}
        {showAppsRow && (
          <DataSet
            title="Apps"
            value={titles.map((t) => getDisplayedSoftwareName(t)).join(", ")}
          />
        )}
        {hasDetailsContent && (
          <RevealButton
            isShowing={showDetails}
            showText="Details"
            hideText="Details"
            caretPosition="after"
            onClick={() => setShowDetails((s) => !s)}
          />
        )}
        {showDetails && scriptResult?.script_contents && (
          <Textarea label="Notification script:" variant="code">
            {scriptResult.script_contents}
          </Textarea>
        )}
        {showDetails && outputBlock && (
          <div className={`${baseClass}__output-section`}>
            <p className={`${baseClass}__output-label`}>
              The{" "}
              <TooltipWrapper
                tipContent="Fleet records the last 10,000 characters to prevent downtime."
                tooltipClass={`${baseClass}__output-tooltip`}
                delayInMs={500}
                position="bottom-start"
              >
                output recorded
              </TooltipWrapper>{" "}
              when ran the script above:
            </p>
            <Textarea variant="code">{outputBlock}</Textarea>
          </div>
        )}
      </div>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Details"
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
