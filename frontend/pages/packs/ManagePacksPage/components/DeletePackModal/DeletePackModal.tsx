import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-pack-modal";

interface IDeletePackModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const DeletePackModal = ({
  onCancel,
  onSubmit,
}: IDeletePackModalProps): JSX.Element => {
  return (
    <Modal
      title={"Delete pack"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to delete the selected packs?
        <div className="modal-cta-wrap">
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse-alert"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="alert"
            onClick={onSubmit}
          >
            Delete
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeletePackModal;
