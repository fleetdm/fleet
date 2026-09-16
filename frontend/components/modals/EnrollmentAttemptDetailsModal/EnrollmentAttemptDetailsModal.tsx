import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import IconStatusMessage from "components/IconStatusMessage";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import { EnrollmentRejectedReason } from "interfaces/activity";
import { FLEET_FLEETD_CONFIG_PROFILE_DISPLAY_NAME } from "interfaces/mdm";
import { timeAgo } from "utilities/date_format";

const baseClass = "enrollment-attempt-details-modal";

export interface IEnrollmentAttemptDetailsModalProps {
  hostDisplayName?: string;
  reason?: EnrollmentRejectedReason | string;
  createdAt?: string;
  onDone: () => void;
}

export const getEnrollmentRejectedReasonText = (reason?: string) => {
  switch (reason) {
    case "one_time_secret_spent":
      return `The host's one-time enroll secret was already used. Resend the "${FLEET_FLEETD_CONFIG_PROFILE_DISPLAY_NAME}" profile to issue a new one.`;
    case "one_time_secret_identifier_mismatch":
      return "A one-time enroll secret was presented with a different serial number or hardware UUID than it was issued for.";
    case "shared_secret_for_mdm_managed_host":
      return "A global or fleet-level enroll secret was used for a host that is enrolled in Fleet MDM or assigned to Fleet in Apple Business.";
    default:
      return "The enroll secret presented was not valid for this host.";
  }
};

const EnrollmentAttemptDetailsModal = ({
  hostDisplayName,
  reason,
  createdAt,
  onDone,
}: IEnrollmentAttemptDetailsModalProps) => {
  const displayTime = createdAt
    ? ` (${timeAgo(new Date(createdAt), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : "";

  return (
    <Modal
      className={baseClass}
      width="large"
      title="Enrollment attempt details"
      onExit={onDone}
    >
      <>
        <IconStatusMessage
          className={`${baseClass}__status-message`}
          iconName="error"
          message={
            <span>
              Fleet rejected an enrollment attempt for{" "}
              <b>{hostDisplayName || "a host"}</b>
              {displayTime}.
            </span>
          }
        />
        <p>
          {getEnrollmentRejectedReasonText(reason)} For assistance, reach out to{" "}
          <CustomLink
            text="Fleet support"
            url="https://fleetdm.com/support"
            newTab
          />
          .
        </p>
        <ModalFooter primaryButtons={<Button onClick={onDone}>Close</Button>} />
      </>
    </Modal>
  );
};

export default EnrollmentAttemptDetailsModal;
