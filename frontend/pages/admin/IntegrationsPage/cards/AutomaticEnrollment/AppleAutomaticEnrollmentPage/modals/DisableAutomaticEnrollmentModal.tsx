import React, { useCallback, useState } from "react";

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
  const [isDeleting, setIsDeleting] = useState(false);

  const onClickConfirm = useCallback(() => {
    setIsDeleting(true);
    onConfirm();
  }, [onConfirm]);

  return (
    <Modal
      title="Disable macOS automatic enrollment"
      onExit={onCancel}
      className={baseClass}
    >
      <div className={baseClass}>
        New macOS hosts won’t automatically enroll to Fleet. If you want to
        enable automatic enrollment, you’ll have to upload a new token.{" "}
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onClickConfirm}
            disabled={isDeleting}
            isLoading={isDeleting}
          >
            Disable
          </Button>
          <Button
            onClick={onCancel}
            disabled={isDeleting}
            variant="inverse-alert"
          >
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DisableAutomaticEnrollmentModal;
