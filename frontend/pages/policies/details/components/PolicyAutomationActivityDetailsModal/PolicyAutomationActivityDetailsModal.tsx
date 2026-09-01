import React from "react";

import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";
import PATHS from "router/paths";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import DataSet from "components/DataSet";
import Textarea from "components/Textarea";
import Icon from "components/Icon";
import { SKIPPED_PRE_INSTALL_OUTPUT } from "components/ActivityDetails/InstallDetails/constants";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

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
  const detailOutput = getDetailOutputText(activity);
  const isSoftwareInstall = activity.type === ActivityType.InstalledSoftware;

  // A code-style output block with a copy button. Renders nothing when empty.
  const renderOutputSection = (label: string, value: string | null) =>
    value ? (
      <Textarea
        key={label}
        variant="code"
        label={
          <div className={`${baseClass}__details-label`}>
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
        {isSoftwareInstall ? (
          <>
            {renderOutputSection(
              "Pre-install query output",
              activity.details?.skipped_install
                ? SKIPPED_PRE_INSTALL_OUTPUT
                : activity.pre_install_output
            )}
            {renderOutputSection("Details", activity.output)}
            {renderOutputSection(
              "Post-install script output",
              activity.post_install_output
            )}
          </>
        ) : (
          renderOutputSection("Details", detailOutput || null)
        )}
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
