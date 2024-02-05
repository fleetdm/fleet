import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-pack-modal";

interface IDeletePackModalProps {
  onCancel: () => void;
  onSubmit: () => void;
  isUpdatingPack: boolean;
}

const DeletePackModal = ({
  onCancel,
  onSubmit,
  isUpdatingPack,
}: IDeletePackModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete pack"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to delete the selected packs?
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onSubmit}
            className="delete-loading"
            isLoading={isUpdatingPack}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeletePackModal;
