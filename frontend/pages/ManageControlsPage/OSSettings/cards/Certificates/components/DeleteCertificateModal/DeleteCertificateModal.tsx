import React, { useContext, useState } from "react";

import certAPI, { ICertificate } from "services/entities/certificates";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-cert-template-modal";

interface IDeleteCertModalProps {
  cert: ICertificate;
  onSuccess: () => void;
  onExit: () => void;
}

const DeleteCertificateModal = ({
  cert,
  onSuccess,
  onExit,
}: IDeleteCertModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);

  const { name, id } = cert;

  const onDelete = async () => {
    setIsUpdating(true);
    try {
      await certAPI.deleteCert(id);
      renderFlash("success", "Successfully deleted certificate.");
      setIsUpdating(false);
      onSuccess();
      onExit();
    } catch (e) {
      setIsUpdating(false);
      renderFlash("error", "Couldn't delete certificate. Please try again.");
    }
  };

  return (
    <Modal className={baseClass} title="Delete certificate" onExit={onExit}>
      <>
        <p>
          This action will remove the <b>{name}</b> certificate from all hosts
          assigned to this team.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onDelete}
            isLoading={isUpdating}
            disabled={isUpdating}
          >
            Delete
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteCertificateModal;
