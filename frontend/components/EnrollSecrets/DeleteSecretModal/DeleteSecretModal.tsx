import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

interface IDeleteSecretModal {
  onDeleteSecret: () => void;
  toggleDeleteSecretModal: () => void;
  isUpdatingSecret: boolean;
}

const baseClass = "delete-secret-modal";

const DeleteSecretModal = ({
  onDeleteSecret,
  toggleDeleteSecretModal,
  isUpdatingSecret,
}: IDeleteSecretModal): JSX.Element => {
  return (
    <Modal
      onExit={toggleDeleteSecretModal}
      onEnter={onDeleteSecret}
      title="Delete secret"
      className={baseClass}
    >
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          <p>Hosts can no longer enroll using this secret.</p>
          <p>
            <CustomLink
              url="https://fleetdm.com/learn-more-about/rotating-enroll-secrets"
              text="Learn about rotating enroll secrets"
              newTab
            />
          </p>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDeleteSecret}
            className="delete-loading"
            isLoading={isUpdatingSecret}
          >
            Delete
          </Button>
          <Button onClick={toggleDeleteSecretModal} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteSecretModal;
