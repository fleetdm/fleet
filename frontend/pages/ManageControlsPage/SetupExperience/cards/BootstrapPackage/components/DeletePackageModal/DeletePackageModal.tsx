import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface DeletePackageModalProps {
  onCancel: () => void;
  onDelete: () => void;
}

const baseClass = "delete-package-modal";

const DeletePackageModal = ({
  onCancel,
  onDelete,
}: DeletePackageModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete bootstrap package"
      onExit={onCancel}
      onEnter={() => onDelete()}
    >
      <>
        <p>Delete the bootstrap package to upload a new one.</p>
        <p>
          If you need to remove the package from macOS hosts already enrolled,
          use your configuration management tool (ex. Munki, Chef, or Puppet).
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={() => onDelete()} variant="alert">
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeletePackageModal;
