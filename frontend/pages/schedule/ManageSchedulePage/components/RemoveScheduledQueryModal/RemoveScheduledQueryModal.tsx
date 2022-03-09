import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-scheduled-query-modal";

interface IRemoveScheduledQueryModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const RemoveScheduledQueryModal = ({
  onCancel,
  onSubmit,
}: IRemoveScheduledQueryModalProps): JSX.Element => {
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
    <Modal
      title={"Remove scheduled query"}
      onExit={onCancel}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to remove the selected queries from the schedule?
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

export default RemoveScheduledQueryModal;
