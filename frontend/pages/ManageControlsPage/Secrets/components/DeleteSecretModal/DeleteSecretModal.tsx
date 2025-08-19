import React, { useContext, useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ISecret } from "interfaces/secrets";
import { NotificationContext } from "context/notification";

import secretsAPI from "services/entities/secrets";

interface DeleteSecretModalProps {
  secret: ISecret | undefined;
  onCancel: () => void;
  onDelete: () => void;
}

const baseClass = "fleet-delete-secret-modal";

const DeleteSecretModal = ({
  secret,
  onCancel,
  onDelete,
}: DeleteSecretModalProps) => {
  const [isDeleting, setIsDeleting] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const onClickDelete = async () => {
    if (!secret) {
      return;
    }
    setIsDeleting(true);
    try {
      await secretsAPI.deleteSecret(secret.id);
      onDelete();
    } catch (error) {
      renderFlash(
        "error",
        "An error occurred while deleting the secret. Please try again."
      );
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal
      title="Delete custom variable?"
      onExit={onCancel}
      className={baseClass}
    >
      <>
        <p>
          This will delete the <b>{secret?.name}</b> custom variable.
        </p>
        <p>
          If this custom variable is used in any configuration profiles or
          scripts, they will fail.
          <br />
          To resolve, edit the configuration profile or script.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onClickDelete}
            isLoading={isDeleting}
            disabled={isDeleting}
          >
            Delete
          </Button>
          <Button variant="inverse-alert" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteSecretModal;
