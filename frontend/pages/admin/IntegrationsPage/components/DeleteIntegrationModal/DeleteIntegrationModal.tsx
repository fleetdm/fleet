import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-integration-modal";

interface IDeleteIntegrationModalProps {
  url: string;
  onSubmit: () => void;
  onCancel: () => void;
}

const DeleteIntegrationModal = ({
  url,
  onSubmit,
  onCancel,
}: IDeleteIntegrationModalProps): JSX.Element => {
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
    <Modal title={"Delete integration"} onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <p>
          This action will delete the{" "}
          <span className={`${baseClass}__url`}>{url}</span> integration.
        </p>
        <p>The automations that use this integration will be turned off.</p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            onClick={onSubmit}
            variant="alert"
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
      </form>
    </Modal>
  );
};

export default DeleteIntegrationModal;
