import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-pack-modal";

interface IRemovePackModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const RemovePackModal = ({
  onCancel,
  onSubmit,
}: IRemovePackModalProps): JSX.Element => {
  return (
    <Modal title={"Delete pack"} onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        Are you sure you want to delete the selected packs?
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
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

export default RemovePackModal;
