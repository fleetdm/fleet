import React from "react";

import { IPolicyAutomationActivity } from "interfaces/policy";
import PATHS from "router/paths";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import DataSet from "components/DataSet";
import Textarea from "components/Textarea";
import Icon from "components/Icon";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import {
  getAutomationRunDisplayName,
  getAutomationStatusIconName,
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
  const { status, created_at, host_id, host_display_name } = activity;
  const detailOutput = getDetailOutputText(activity);

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
              <Icon name={getAutomationStatusIconName(status)} />
              {getAutomationRunDisplayName(activity)}
            </span>
          }
        />
        {detailOutput && (
          <Textarea
            variant="code"
            label={
              <div className={`${baseClass}__details-label`}>
                <span>Details</span>
                <CopyButton
                  copyText={detailOutput}
                  size="small"
                  ariaLabel="Copy details"
                />
              </div>
            }
          >
            {detailOutput}
          </Textarea>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
          {onResetPolicy && (
            <Button
              variant="inverse"
              onClick={onResetPolicy}
              className={`${baseClass}__reset`}
            >
              <Icon name="refresh" />
              Reset policy
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
};

export default PolicyAutomationActivityDetailsModal;
