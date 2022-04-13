import React, { useEffect } from "react";

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
    <Modal title={"Delete query"} onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        Are you sure you want to delete the selected queries?
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse-alert"
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

export default RemoveQueryModal;
