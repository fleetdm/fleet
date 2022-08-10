import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-label-modal";

interface IDeleteLabelModalProps {
  onSubmit: () => void;
  onCancel: () => void;
  isUpdatingLabel: boolean;
}

const DeleteLabelModal = ({
  onSubmit,
  onCancel,
  isUpdatingLabel,
}: IDeleteLabelModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete label"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <p>Are you sure you wish to delete this label?</p>
        <div className="modal-cta-wrap">
          <Button
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdatingLabel}
          >
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

export default DeleteLabelModal;
