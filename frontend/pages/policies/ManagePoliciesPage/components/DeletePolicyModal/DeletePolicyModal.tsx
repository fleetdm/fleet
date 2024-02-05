import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-policy-modal";

interface IDeletePolicyModalProps {
  isUpdatingPolicies: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const DeletePolicyModal = ({
  isUpdatingPolicies,
  onCancel,
  onSubmit,
}: IDeletePolicyModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete policies"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to delete the selected policies?
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onSubmit}
            className="delete-loading"
            isLoading={isUpdatingPolicies}
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

export default DeletePolicyModal;
