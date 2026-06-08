import React, { useState } from "react";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import IconStatusMessage from "components/IconStatusMessage";
import Textarea from "components/Textarea";
import RevealButton from "components/buttons/RevealButton";

const baseClass = "certificate-install-details-modal";

export interface ICertificateInstallDetails {
  certificateName: string;
  hostDisplayName: string;
  status: string;
  detail: string;
}

interface ICertificateInstallDetailsModalProps {
  details: ICertificateInstallDetails;
  onCancel: () => void;
}

const CertificateInstallDetailsModal = ({
  details,
  onCancel,
}: ICertificateInstallDetailsModalProps) => {
  const { certificateName, hostDisplayName, detail } = details;

  const [showDetails, setShowDetails] = useState(false);

  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";

  return (
    <Modal
      title="Install details"
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
              Fleet failed to install certificate <b>{certificateName}</b> on{" "}
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

export default CertificateInstallDetailsModal;
