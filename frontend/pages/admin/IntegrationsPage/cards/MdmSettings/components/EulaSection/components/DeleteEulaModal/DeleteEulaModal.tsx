import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-eula-modal";

interface IDeleteEulaModalProps {
  onDelete: () => void;
  onCancel: () => void;
}

const DeleteEulaModal = ({ onDelete, onCancel }: IDeleteEulaModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete EULA"
      onExit={onCancel}
      onEnter={() => onDelete()}
    >
      <>
        <p>
          End users won’t be required to agree to this EULA on macOS hosts that
          automatically enroll.
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

export default DeleteEulaModal;
