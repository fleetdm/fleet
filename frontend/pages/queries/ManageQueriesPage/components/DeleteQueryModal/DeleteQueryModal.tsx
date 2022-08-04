import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "delete-query-modal";

interface IDeleteQueryModalProps {
  isLoading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const DeleteQueryModal = ({
  isLoading,
  onCancel,
  onSubmit,
}: IDeleteQueryModalProps): JSX.Element => {
  return (
    <Modal
      title={"Delete query"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        {isLoading ? (
          <Spinner />
        ) : (
          <div className={baseClass}>
            Are you sure you want to delete the selected queries?
            <div className="modal-cta-wrap">
              <Button onClick={onCancel} variant="inverse-alert">
                Cancel
              </Button>
              <Button type="button" variant="alert" onClick={onSubmit}>
                Delete
              </Button>
            </div>
          </div>
        )}
      </>
    </Modal>
  );
};

export default DeleteQueryModal;
