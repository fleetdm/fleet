import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-policy-modal";

interface IDeletePoliciesModalProps {
  isUpdatingPolicies: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const DeletePoliciesModal = ({
  isUpdatingPolicies, // shared state from parent, not only for deletes
  onCancel,
  onSubmit,
}: IDeletePoliciesModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete policies"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Deleting these policies will disable any associated automations, such as
        automatic software install or automatic script run.
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

export default DeletePoliciesModal;
