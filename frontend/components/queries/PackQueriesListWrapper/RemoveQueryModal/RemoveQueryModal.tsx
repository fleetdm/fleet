import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-query-modal";

interface IRemoveQueryModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const RemoveQueryModal = ({
  onCancel,
  onSubmit,
}: IRemoveQueryModalProps): JSX.Element => {
  return (
    <Modal title={"Remove query"} onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        Are you sure you want to remove the selected queries from your pack?
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

export default RemoveQueryModal;
