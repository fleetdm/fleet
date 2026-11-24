import React, { useState } from "react";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import IconStatusMessage from "components/IconStatusMessage";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Textarea from "components/Textarea";
import { dateAgo } from "utilities/date_format";

const baseClass = "rotation-failed-details-modal";

export type RotationFailedSubject =
  | "managed local account password"
  | "disk encryption key";

interface IRotationFailedDetailsModalProps {
  /** The reason Fleet recorded for the failure. */
  detail: string;
  hostDisplayName: string;
  /** @default "managed local account password" */
  subject?: RotationFailedSubject;
  /** When set, the failure's relative time follows the sentence. */
  createdAt?: string;
  onCancel: () => void;
}

const RotationFailedDetailsModal = ({
  detail,
  hostDisplayName,
  subject = "managed local account password",
  createdAt,
  onCancel,
}: IRotationFailedDetailsModalProps) => {
  const [showDetails, setShowDetails] = useState(false);

  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";
  const preposition = subject === "disk encryption key" ? "for" : "on";

  return (
    <Modal
      title="Rotation details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__modal-content`}>
        <IconStatusMessage
          className={`${baseClass}__status-message`}
          iconName="error"
          message={
            <span>
              Fleet failed to rotate the {subject} {preposition} {formattedHost}
              {createdAt && ` (${dateAgo(createdAt)})`}.
              {subject === "disk encryption key" &&
                " Ask the end user to log out of their device or restart it. They'll see instructions on their My device page."}
            </span>
          }
        />
        {detail && (
          <>
            <RevealButton
              isShowing={showDetails}
              showText="Details"
              hideText="Details"
              caretPosition="after"
              onClick={() => setShowDetails((prev) => !prev)}
            />
            {showDetails && (
              <Textarea label="Error details:" variant="code">
                {detail}
              </Textarea>
            )}
          </>
        )}
      </div>
      <ModalFooter primaryButtons={<Button onClick={onCancel}>Close</Button>} />
    </Modal>
  );
};

export default RotationFailedDetailsModal;
