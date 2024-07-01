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

  const onClickConfirm: React.MouseEventHandler<HTMLButtonElement> = useCallback(() => {
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
        New macOS hosts won’t automatically enroll to Fleet.
        <br />
        <br />
        If you want to enable automatic enrollment, you’ll have to create a new
        MDM server in Apple Business Manager, reassign all devices, and upload
        your new server token in Fleet.{" "}
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
