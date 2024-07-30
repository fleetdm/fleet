import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-integration-modal";

interface IDeleteIntegrationModalProps {
  url: string;
  projectKey: string;
  onSubmit: () => void;
  onCancel: () => void;
  isUpdatingIntegration: boolean;
}

const DeleteIntegrationModal = ({
  url,
  projectKey,
  onSubmit,
  onCancel,
  isUpdatingIntegration,
}: IDeleteIntegrationModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete integration"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <form className={`${baseClass}__form`}>
        <p>
          This action will delete the{" "}
          <span className={`${baseClass}__url`}>
            {url} - {projectKey}
          </span>{" "}
          integration.
        </p>
        <p>The automations that use this integration will be turned off.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdatingIntegration}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default DeleteIntegrationModal;
