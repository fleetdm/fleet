import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-query-modal";

interface IDeleteQueryModalProps {
  isUpdatingQueries: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const DeleteQueryModal = ({
  isUpdatingQueries,
  onCancel,
  onSubmit,
}: IDeleteQueryModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete query"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to delete the selected queries?
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onSubmit}
            className="delete-loading"
            isLoading={isUpdatingQueries}
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

export default DeleteQueryModal;
