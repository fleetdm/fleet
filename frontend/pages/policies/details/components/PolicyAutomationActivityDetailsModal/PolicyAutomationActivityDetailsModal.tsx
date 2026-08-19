import React, { useState } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";
import PATHS from "router/paths";
import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import DataSet from "components/DataSet";
import Textarea from "components/Textarea";
import Icon from "components/Icon";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import {
  getCaveatSentence,
  getAutomationNotifiedSentence,
  isNotifyBeforePatchingSkip,
  SKIPPED_INSTALL_NOTIFY_EXPLANATION,
  EXIT_CODES_NEEDING_EUE_LINK,
  PATCHING_END_USER_EXPERIENCE_URL,
} from "components/ActivityDetails/NotifyBeforePatchingDetailsModal/helpers";

import {
  getAutomationRunDisplayName,
  getAutomationStatusIcon,
  getDetailOutputText,
} from "../PolicyAutomationsActivitiesTable/helpers";

const baseClass = "policy-automation-activity-details-modal";

interface IPolicyAutomationActivityDetailsModalProps {
  activity: IPolicyAutomationActivity;
  onCancel: () => void;
  /** When provided, renders a "Reset policy" action in the footer. */
  onResetPolicy?: () => void;
}

const PolicyAutomationActivityDetailsModal = ({
  activity,
  onCancel,
  onResetPolicy,
}: IPolicyAutomationActivityDetailsModalProps): JSX.Element => {
  const { created_at, host_id, host_display_name } = activity;
  const isNotify = activity.type === ActivityType.NotifiedEndUserBeforePatching;
  const isSoftwareInstall = activity.type === ActivityType.InstalledSoftware;
  const isSkippedInstall =
    isSoftwareInstall && !!activity.details?.skipped_install;
  const isSkippedNotifyVariant =
    isSkippedInstall && isNotifyBeforePatchingSkip(activity.pre_install_output);
  const scriptExecutionId = activity.details?.script_execution_id;

  const [showDetails, setShowDetails] = useState(false);

  // Only the notify branches need the exit code (not on the activity itself).
  const { data: scriptResult } = useQuery<IScriptResultResponse, AxiosError>(
    ["notify-script-result", scriptExecutionId],
    () => scriptsAPI.getScriptResult(scriptExecutionId as string),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isNotify && !!scriptExecutionId,
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
    }
  );

  const detailOutput = getDetailOutputText(activity);

  const renderProseSentence = (): string | null => {
    if (isNotify) {
      if (activity.status === "success") {
        return getAutomationNotifiedSentence(activity.details?.time_before);
      }
      return getCaveatSentence(scriptExecutionId, scriptResult?.exit_code);
    }
    if (isSkippedNotifyVariant) {
      return SKIPPED_INSTALL_NOTIFY_EXPLANATION;
    }
    return null;
  };

  const proseSentence = renderProseSentence();
  const showEueLink =
    isNotify &&
    scriptResult?.exit_code != null &&
    EXIT_CODES_NEEDING_EUE_LINK.has(scriptResult.exit_code);

  // Notify + notify-skip cases hide output behind a Details reveal.
  const revealLabel = (() => {
    if (isNotify) return "Notification script output:";
    if (isSkippedNotifyVariant) return "Pre-install query output:";
    return null;
  })();
  const revealOutput = (() => {
    if (isNotify) return scriptResult?.output || activity.output || null;
    if (isSkippedNotifyVariant) return activity.pre_install_output;
    return null;
  })();

  // A code-style output block with a copy button. Renders nothing when empty.
  // `plainLabel` drops the bold treatment for labels inside the Details reveal.
  const renderOutputSection = (
    label: string,
    value: string | null,
    plainLabel = false
  ) =>
    value ? (
      <Textarea
        key={label}
        variant="code"
        label={
          <div
            className={
              plainLabel
                ? `${baseClass}__plain-label`
                : `${baseClass}__details-label`
            }
          >
            <span>{label}</span>
            <CopyButton
              copyText={value}
              size="small"
              ariaLabel={`Copy ${label.toLowerCase()}`}
            />
          </div>
        }
      >
        {value}
      </Textarea>
    ) : null;

  const renderBody = () => {
    if (proseSentence) {
      return (
        <>
          <p className={`${baseClass}__prose`}>
            {proseSentence}
            {showEueLink && (
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
          {revealOutput && revealLabel && (
            <>
              <RevealButton
                isShowing={showDetails}
                showText="Details"
                hideText="Details"
                caretPosition="after"
                onClick={() => setShowDetails((s) => !s)}
              />
              {showDetails &&
                renderOutputSection(revealLabel, revealOutput, true)}
            </>
          )}
        </>
      );
    }
    if (isSoftwareInstall) {
      return (
        <>
          {renderOutputSection(
            "Pre-install query output",
            activity.pre_install_output
          )}
          {renderOutputSection("Details", activity.output)}
          {renderOutputSection(
            "Post-install script output",
            activity.post_install_output
          )}
        </>
      );
    }
    return renderOutputSection("Details", detailOutput || null);
  };

  return (
    <Modal title="Details" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__modal-content`}>
        <div className={`${baseClass}__row`}>
          <DataSet
            title="Host"
            value={
              host_display_name ? (
                <CustomLink
                  url={PATHS.HOST_DETAILS(host_id)}
                  text={host_display_name}
                />
              ) : (
                "---"
              )
            }
          />
          <DataSet
            title="Time"
            value={<HumanTimeDiffWithDateTip timeString={created_at} />}
          />
        </div>
        <DataSet
          title="Status"
          value={
            <span className={`${baseClass}__status`}>
              <Icon
                name={getAutomationStatusIcon(activity).name}
                color={getAutomationStatusIcon(activity).color}
              />
              {getAutomationRunDisplayName(activity)}
            </span>
          }
        />
        {renderBody()}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
          {onResetPolicy && (
            <Button
              variant="secondary"
              onClick={onResetPolicy}
              className={`${baseClass}__reset`}
              icon="refresh"
            >
              Reset policy
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
};

export default PolicyAutomationActivityDetailsModal;
