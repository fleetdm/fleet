import React, { useContext, useState } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";
import { getErrorReason } from "interfaces/errors";
import certificatesAPI from "services/entities/certificates";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-certificate-authority-modal";

interface IDeleteCertificateAuthorityModalProps {
  certAuthority: ICertificateAuthorityPartial;
  onExit: () => void;
}

const DeleteCertificateAuthorityModal = ({
  certAuthority,
  onExit,
}: IDeleteCertificateAuthorityModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);

  const onDeleteCertAuthority = async () => {
    setIsUpdating(true);
    try {
      await certificatesAPI.deleteCertificateAuthority(certAuthority.id);
      renderFlash(
        "success",
        "Successfully deleted your certificate authority."
      );
      setIsUpdating(false);
      onExit();
    } catch (e) {
      setIsUpdating(false);
      const status = (e as { status?: number })?.status;
      const reason = status === 409 ? getErrorReason(e) : "";
      renderFlash(
        "error",
        reason || "Couldn't delete certificate authority. Please try again."
      );
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Delete certificate authority (CA)"
      onExit={onExit}
    >
      <p>
        Fleet won&apos;t remove certificates from the certificate authority (
        <b>{certAuthority.name}</b>) on existing hosts.
      </p>
      <div className="modal-cta-wrap">
        <Button
          variant="alert"
          onClick={onDeleteCertAuthority}
          isLoading={isUpdating}
          disabled={isUpdating}
        >
          Delete
        </Button>
        <Button variant="inverse-alert" onClick={onExit}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteCertificateAuthorityModal;
