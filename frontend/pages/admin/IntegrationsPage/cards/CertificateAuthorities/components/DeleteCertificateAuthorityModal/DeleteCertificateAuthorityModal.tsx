import React from "react";

import { ICertificateIntegration } from "interfaces/integration";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-certificate-authority-modal";

interface IDeleteCertificateAuthorityModalProps {
  certAuthority: ICertificateIntegration;
  onExit: () => void;
}

const DeleteCertificateAuthorityModal = ({
  certAuthority,
  onExit,
}: IDeleteCertificateAuthorityModalProps) => {
  const onDeleteCertAuthority = () => {
    console.log("Delete certificate authority", certAuthority);
  };

  return (
    <Modal
      className={baseClass}
      title="Delete certificate authority (CA)"
      onExit={onExit}
    >
      <>
        <p>
          Fleet won&apos;t remove certificates from the certificate authority (
          <b>SCEP_WIFI</b>) on existing hosts.
        </p>
        <div className="modal-cta-wrap">
          <Button variant="alert" onClick={onDeleteCertAuthority}>
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

export default DeleteCertificateAuthorityModal;
