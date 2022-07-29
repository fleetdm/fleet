import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-policies-modal";

interface IDeletePoliciesModalProps {
  isLoading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const DeletePoliciesModal = ({
  isLoading,
  onCancel,
  onSubmit,
}: IDeletePoliciesModalProps): JSX.Element => {
  return (
    <Modal
      title={"Delete policies"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        {isLoading ? (
          <Spinner />
        ) : (
          <div className={baseClass}>
            Are you sure you want to delete the selected policies?
            <div className={`${baseClass}__btn-wrap`}>
              <Button
                className={`${baseClass}__btn`}
                type="button"
                variant="alert"
                onClick={onSubmit}
              >
                Delete
              </Button>
              <Button
                className={`${baseClass}__btn`}
                onClick={onCancel}
                variant="inverse-alert"
              >
                Cancel
              </Button>
            </div>
          </div>
        )}
      </>
    </Modal>
  );
};

export default DeletePoliciesModal;
