import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-scheduled-query-modal";

interface IRemoveScheduledQueryModalProps {
  scheduleIsRemoving: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const RemoveScheduledQueryModal = ({
  scheduleIsRemoving,
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
      {scheduleIsRemoving ? (
        <Spinner />
      ) : (
        <div className={baseClass}>
          Are you sure you want to remove the selected queries from the
          schedule?
          <div className="modal-cta-wrap">
            <Button onClick={onCancel} variant="inverse-alert">
              Cancel
            </Button>
            <Button type="button" variant="alert" onClick={onSubmit}>
              Remove
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
};

export default RemoveScheduledQueryModal;
