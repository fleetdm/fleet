import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-policies-modal";

interface IRemovePoliciesModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const RemovePoliciesModal = ({
  onCancel,
  onSubmit,
}: IRemovePoliciesModalProps): JSX.Element => {
  return (
    <Modal title={"Delete policies"} onExit={onCancel} className={baseClass}>
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
    </Modal>
  );
};

export default RemovePoliciesModal;
