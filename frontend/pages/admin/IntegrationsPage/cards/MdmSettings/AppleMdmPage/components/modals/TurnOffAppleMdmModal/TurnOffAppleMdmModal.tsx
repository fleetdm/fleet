import React, { useCallback, useState } from "react";

import Button from "components/buttons/Button";

import Modal from "components/Modal";

const baseClass = "modal turn-off-apple-mdm-modal";

interface ITurnOffAppleMdmModalProps {
  onCancel: () => void;
  onConfirm: () => void;
}

const TurnOffAppleMdmModal = ({
  onConfirm,
  onCancel,
}: ITurnOffAppleMdmModalProps): JSX.Element => {
  const [isDeleting, setIsDeleting] = useState(false);

  const onClickConfirm = useCallback(() => {
    setIsDeleting(true);
    onConfirm();
  }, [onConfirm]);

  return (
    <Modal title="Turn off MDM" onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        If you want to use MDM features again, you&apos;ll have to upload a new
        APNs certificate and all end users will have to turn MDM off and back
        on.
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onClickConfirm}
            isLoading={isDeleting}
            disabled={isDeleting}
          >
            Turn off
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

export default TurnOffAppleMdmModal;
