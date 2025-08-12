import React, { useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ISecret } from "interfaces/secrets";

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
  return (
    <Modal title="Add custom variable" onExit={onCancel} className={baseClass}>
      <>
        <p>
          This will delete the {secret?.name} custom variable. If this custom
          variable is used in any configuration profiles or scripts, they will
          fail. To resolve, edit the configuration profile or script.
        </p>
        <div className="modal-cta-wrap">
          <Button variant="alert" onClick={onDelete}>
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
