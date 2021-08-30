import React from "react";

import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-policies-modal";

interface IRemovePoliciesModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const RemovePoliciesModal = (props: IRemovePoliciesModalProps): JSX.Element => {
  const { onCancel, onSubmit } = props;

  return (
    <Modal title={"Remove policies"} onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        Are you sure you want to remove the selected policies?
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="alert"
            onClick={onSubmit}
          >
            Remove
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
