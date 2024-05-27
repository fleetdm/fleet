import React, { useCallback } from "react";

import Button from "components/buttons/Button";

import Modal from "components/Modal";

const baseClass = "modal disable-automatic-enrollment-modal";

interface IDisableAutomaticEnrollmentModalProps {
  onCancel: () => void;
  onConfirm: () => void;
}

const DisableAutomaticEnrollmentModal = ({
  onConfirm,
  onCancel,
}: IDisableAutomaticEnrollmentModalProps): JSX.Element => {
  // TODO: add loading state for the button? Handle submission inside this modal?

  // TODO: confirm button text should be "Delete" rather than "Disable"

  return (
    <Modal
      title="Disable macOS automatic enrollment"
      onExit={onCancel}
      onEnter={onConfirm}
      className={baseClass}
    >
      <div className={baseClass}>
        New macOS hosts won’t automatically enroll to Fleet. If you want to
        enable automatic enrollment, you’ll have to upload a new token.{" "}
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onConfirm}
            // className="delete-loading"
            // isLoading={}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DisableAutomaticEnrollmentModal;
