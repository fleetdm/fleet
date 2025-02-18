import React from "react";

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
        <p>
          If a configuration profile uses this label as a custom target, the
          profile will break. After deleting the label, remove broken profiles
          and upload new profiles in their place.
        </p>
        <p>
          If software uses this label as a custom target, the label will not be
          able to be deleted. Please remove the label from the software target
          first before deleting.
        </p>
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
