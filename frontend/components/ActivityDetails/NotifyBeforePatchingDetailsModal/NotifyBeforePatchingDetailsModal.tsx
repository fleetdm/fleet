import React, { useState } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { IActivityDetails } from "interfaces/activity";
import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { pluralize } from "utilities/strings/stringUtils";
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
  getCaveatSentence,
  EXIT_CODES_NEEDING_EUE_LINK,
  PATCHING_END_USER_EXPERIENCE_URL,
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

  const timeLabel = timeBefore === 300 ? "5 minutes" : "1 hour";
  const failed = status === "failed";

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

  const caveatSentence = getCaveatSentence(
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
    // Nothing to reveal for a dispatcher-caught deferral (no script ran).
    const hasRevealable =
      !!scriptResult?.script_contents || scriptResult?.exit_code != null;
    const outputBlock =
      scriptResult?.exit_code != null
        ? `Exit code: ${scriptResult.exit_code}\n${scriptResult.output ?? ""}`
        : null;

    // Bold titles, Oxford comma, ", and N more app(s)" past three.
    const bold = (name: string) => <b>{getDisplayedSoftwareName(name)}</b>;
    const overflow = titles.length - 3;
    let titleList: React.ReactNode = null;
    if (titles.length === 1) {
      titleList = bold(titles[0]);
    } else if (titles.length === 2) {
      titleList = (
        <>
          {bold(titles[0])} and {bold(titles[1])}
        </>
      );
    } else if (titles.length === 3) {
      titleList = (
        <>
          {bold(titles[0])}, {bold(titles[1])}, and {bold(titles[2])}
        </>
      );
    } else if (titles.length > 3) {
      titleList = (
        <>
          {bold(titles[0])}, {bold(titles[1])}, {bold(titles[2])}, and{" "}
          {overflow} more {pluralize(overflow, "app")}
        </>
      );
    }

    // Apps row is redundant when the intro already lists everything (≤3 apps).
    const showAppsRow = titles.length > 3;

    return (
      <div className={`${baseClass}__content`}>
        <IconStatusMessage
          className={`${baseClass}__status-message`}
          iconName={failed ? "error-outline" : "success-outline"}
          message={
            <span>
              Fleet {verb} end user {timeLabel} before patching
              {titleList && <> {titleList}</>} on{" "}
              <b>{hostName || "this host"}</b>.
            </span>
          }
        />
        {caveatSentence && (
          <p className={`${baseClass}__caveat`}>
            {caveatSentence}
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
        {hasRevealable && (
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
          <Textarea
            label={
              <>
                The{" "}
                <TooltipWrapper
                  tipContent="Fleet records the last 10,000 characters to prevent downtime."
                  tooltipClass={`${baseClass}__output-tooltip`}
                  delayInMs={500}
                >
                  output recorded
                </TooltipWrapper>{" "}
                when ran the script above:
              </>
            }
            variant="code"
          >
            {outputBlock}
          </Textarea>
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
