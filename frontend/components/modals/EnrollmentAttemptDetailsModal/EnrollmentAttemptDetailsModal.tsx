import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import IconStatusMessage from "components/IconStatusMessage";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import { EnrollmentRejectedReason } from "interfaces/activity";
import { FLEET_FLEETD_CONFIG_PROFILE_DISPLAY_NAME } from "interfaces/mdm";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { timeAgo } from "utilities/date_format";

const baseClass = "enrollment-attempt-details-modal";

export interface IEnrollmentAttemptDetailsModalProps {
  hostDisplayName?: string;
  /** Fallback identifier when the host has no display name. */
  hostSerial?: string;
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
    case "shared_secret_for_mdm_managed_host":
      return (
        <>
          A shared enroll secret was used for a host that requires a one-time
          enroll secret.{" "}
          <CustomLink
            text="How to troubleshoot"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/enrollment-troubleshooting`}
            newTab
          />
        </>
      );
    case "one_time_secret_identifier_mismatch":
      // Told entirely in the modal headline; there is no body text.
      return null;
    default:
      return "The enroll secret presented was not valid for this host.";
  }
};

const EnrollmentAttemptDetailsModal = ({
  hostDisplayName,
  hostSerial,
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

  // Same fallback chain as the activity feed row: name, then serial, then
  // nothing identifying.
  let host: React.ReactNode = "a host";
  if (hostDisplayName) {
    host = <b>{hostDisplayName}</b>;
  } else if (hostSerial) {
    host = (
      <>
        a host with serial number <b>{hostSerial}</b>
      </>
    );
  }

  // The mismatch rejection is about a different device presenting this
  // host's secret, so the whole story fits in the headline.
  const isIdentifierMismatch = reason === "one_time_secret_identifier_mismatch";

  const supportLink = (
    <CustomLink text="Fleet support" url="https://fleetdm.com/support" newTab />
  );

  let message: React.ReactNode;
  if (!isIdentifierMismatch) {
    message = (
      <span>
        Fleet rejected an enrollment for {host}
        {displayTime}.
      </span>
    );
  } else if (hostDisplayName) {
    message = (
      <span>
        Fleet rejected an enrollment for a host that tried to enroll with{" "}
        <b>{hostDisplayName}&apos;s</b> one-time enroll secret{displayTime}.
        Reach out to {supportLink}.
      </span>
    );
  } else {
    message = (
      <span>
        Fleet rejected an enrollment for a host that tried to enroll with the
        one-time enroll secret issued to {host}
        {displayTime}. Reach out to {supportLink}.
      </span>
    );
  }

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
          message={message}
        />
        {!isIdentifierMismatch && (
          <p>{getEnrollmentRejectedReasonText(reason)}</p>
        )}
        <ModalFooter primaryButtons={<Button onClick={onDone}>Close</Button>} />
      </>
    </Modal>
  );
};

export default EnrollmentAttemptDetailsModal;
