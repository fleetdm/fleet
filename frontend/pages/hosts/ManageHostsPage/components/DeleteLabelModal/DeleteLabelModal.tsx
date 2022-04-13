import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-label-modal";

interface IDeleteLabelModalProps {
  onSubmit: () => void;
  onCancel: () => void;
}

const DeleteLabelModal = ({
  onSubmit,
  onCancel,
}: IDeleteLabelModalProps): JSX.Element => {
  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onSubmit();
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  }, []);

  return (
    <Modal title="Delete label" onExit={onCancel} className={baseClass}>
      <>
        <p>Are you sure you wish to delete this label?</p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
          <Button onClick={onSubmit} variant="alert">
            Delete
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteLabelModal;
