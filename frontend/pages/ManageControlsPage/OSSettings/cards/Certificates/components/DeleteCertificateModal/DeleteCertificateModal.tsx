import React, { useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import certAPI from "services/entities/certificates";

const baseClass = "delete-cert-template-modal";

interface IDeleteCertModalProps {
  cert: {
    id: number;
    name: string;
    isCA?: boolean;
  };
  onSuccess: () => void;
  onExit: () => void;
}

const DeleteCertificateModal = ({
  cert,
  onSuccess,
  onExit,
}: IDeleteCertModalProps) => {
  const [isUpdating, setIsUpdating] = useState(false);

  const { name, id, isCA } = cert;

  const onDelete = async () => {
    setIsUpdating(true);
    try {
      if (isCA) {
        await certAPI.deleteCACert(id);
      } else {
        await certAPI.deleteCert(id);
      }
      notify.success("Successfully deleted certificate.");
      setIsUpdating(false);
      onSuccess();
      onExit();
    } catch (e) {
      setIsUpdating(false);
      notify.error("Couldn't delete certificate. Please try again.", {
        response: e,
      });
    }
  };

  return (
    <Modal className={baseClass} title="Delete certificate" onExit={onExit}>
      <p>
        This action will remove the <b>{name}</b> certificate from all hosts
        assigned to this fleet.
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
        <Button variant="secondary" onClick={onExit}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteCertificateModal;
