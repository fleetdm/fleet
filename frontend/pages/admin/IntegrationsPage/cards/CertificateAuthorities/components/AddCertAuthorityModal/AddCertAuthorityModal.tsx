import React, { useContext, useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";

const baseClass = "add-cert-authority-modal";

interface IAddCertAuthorityModalProps {
  onExit: () => void;
}

const AddCertAuthorityModal = ({ onExit }: IAddCertAuthorityModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);

  const onAddCertAuthority = async () => {
    setIsUpdating(true);
    try {
      renderFlash("success", "Successfully added your certificate authority.");
    } catch (e) {
      renderFlash("error", "test");
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate authority (CA)"
      onExit={onExit}
    >
      <>
        <div className="modal-cta-wrap">
          <Button
            onClick={onAddCertAuthority}
            isLoading={isUpdating}
            disabled={isUpdating}
          >
            Add CA
          </Button>
          <Button variant="text-link" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddCertAuthorityModal;
