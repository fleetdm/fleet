import React, { useState } from "react";

import microsoftGraphCredentialsAPI from "services/entities/microsoft_graph_credentials";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { notify } from "components/ToastNotification";

const baseClass = "delete-microsoft-graph-credential-modal";

interface IDeleteMicrosoftGraphCredentialModalProps {
  onExit: () => void;
  onDeleted: () => void;
}

const DeleteMicrosoftGraphCredentialModal = ({
  onExit,
  onDeleted,
}: IDeleteMicrosoftGraphCredentialModalProps) => {
  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteCredential = async () => {
    setIsDeleting(true);

    try {
      await microsoftGraphCredentialsAPI.deleteCredentials();
      notify.success("Successfully deleted Microsoft Graph credential.");
      // onDeleted unmounts this modal, so the in-flight flag is only reset on the paths that keep it mounted.
      onDeleted();
    } catch (err) {
      notify.error("Couldn't delete Microsoft Graph credential.", {
        response: err,
      });
      setIsDeleting(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Delete credential"
      onExit={onExit}
      width="medium"
      isContentDisabled={isDeleting}
    >
      <p>
        Fleet will stop syncing Windows Autopilot devices from this tenant.
        Devices already synced will remain as pending hosts until they enroll or
        you delete them.
      </p>
      <div className="modal-cta-wrap">
        <Button
          onClick={onDeleteCredential}
          variant="alert"
          isLoading={isDeleting}
        >
          Delete
        </Button>
        <Button onClick={onExit} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteMicrosoftGraphCredentialModal;
