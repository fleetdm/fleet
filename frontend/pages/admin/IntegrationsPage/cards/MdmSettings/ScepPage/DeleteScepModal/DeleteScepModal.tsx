import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-scep-modal";

interface IDeleteScepModalProps {
  onDelete: () => void;
  onCancel: () => void;
}

const DeleteScepModal = ({ onDelete, onCancel }: IDeleteScepModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete SCEP"
      onExit={onCancel}
      onEnter={() => onDelete()}
    >
      <>
        <p>
          {/* End users won’t be required to agree to this SCEP on macOS hosts that
          automatically enroll. */}
          TODO
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

export default DeleteScepModal;
