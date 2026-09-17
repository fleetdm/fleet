import React from "react";

import Button from "components/buttons/Button";
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

export const getEnrollmentRejectedReasonText = (
  reason?: string
): React.ReactNode => {
  switch (reason) {
    case "one_time_secret_spent":
      return (
        <>
          The host&apos;s one-time enroll secret was already used. Resend the{" "}
          <b>{FLEET_FLEETD_CONFIG_PROFILE_DISPLAY_NAME}</b> profile to issue a
          new one from <b>Host details &gt; Controls</b>.
        </>
      );
    case "one_time_secret_identifier_mismatch":
      return "A one-time enroll secret was presented with a different serial number or hardware UUID than it was issued for.";
    case "shared_secret_for_mdm_managed_host":
      return "A shared enroll secret was used for a host that requires a one-time enroll secret.";
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
      title="Enrollment details"
      onExit={onDone}
    >
      <>
        <IconStatusMessage
          className={`${baseClass}__status-message`}
          iconName="error"
          message={
            <span>
              Fleet rejected an enrollment for{" "}
              <b>{hostDisplayName || "a host"}</b>
              {displayTime}.
            </span>
          }
        />
        <p>{getEnrollmentRejectedReasonText(reason)}</p>
        <ModalFooter primaryButtons={<Button onClick={onDone}>Close</Button>} />
      </>
    </Modal>
  );
};

export default EnrollmentAttemptDetailsModal;
