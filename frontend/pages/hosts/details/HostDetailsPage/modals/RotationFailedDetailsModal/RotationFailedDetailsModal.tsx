import React, { useState } from "react";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import IconStatusMessage from "components/IconStatusMessage";
import Textarea from "components/Textarea";
import RevealButton from "components/buttons/RevealButton";

const baseClass = "rotation-failed-details-modal";

interface IRotationFailedDetailsModalProps {
  /** The reason the host reported when it could not set the new password. */
  detail: string;
  hostDisplayName: string;
  onCancel: () => void;
}

const RotationFailedDetailsModal = ({
  detail,
  hostDisplayName,
  onCancel,
}: IRotationFailedDetailsModalProps) => {
  const [showDetails, setShowDetails] = useState(false);

  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";

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
              Fleet failed to rotate the managed local account password on{" "}
              {formattedHost}.
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
