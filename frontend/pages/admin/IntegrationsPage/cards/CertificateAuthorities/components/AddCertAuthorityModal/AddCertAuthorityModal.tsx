import React, { useContext, useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";
import DigicertForm from "../DigicertForm";

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
    setIsUpdating(false);
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate authority (CA)"
      width="large"
      onExit={onExit}
    >
      <>
        <DigicertForm
          submitBtnText="Add CA"
          onSubmit={onAddCertAuthority}
          onCancel={onExit}
        />
      </>
    </Modal>
  );
};

export default AddCertAuthorityModal;
