import React, { useContext, useState } from "react";

import {
  ICertificateAuthorityType,
  ICertificateIntegration,
} from "interfaces/integration";
import certificatesAPI from "services/entities/certificates";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import {
  generateCertAuthorityDisplayName,
  useCertAuthorityDataGenerator,
} from "./helpers";

const baseClass = "delete-certificate-authority-modal";

interface IDeleteCertificateAuthorityModalProps {
  listItemId: string;
  certAuthority: ICertificateIntegration;
  onExit: () => void;
}

const DeleteCertificateAuthorityModal = ({
  listItemId,
  certAuthority,
  onExit,
}: IDeleteCertificateAuthorityModalProps) => {
  const certAuthorityType = listItemId.split(
    "-"
  )[0] as ICertificateAuthorityType;

  const { generateDeletePatchData } = useCertAuthorityDataGenerator(
    certAuthorityType,
    certAuthority
  );
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);

  const onDeleteCertAuthority = async () => {
    setIsUpdating(true);
    try {
      const newConfig = await certificatesAPI.deleteCertificateAuthority(
        generateDeletePatchData()
      );
      renderFlash(
        "success",
        "Successfully deleted your certificate authority."
      );
      setConfig(newConfig);
    } catch (e) {
      renderFlash(
        "error",
        "Couldn't delete certificate authority. Please try again."
      );
    }
    setIsUpdating(false);
    onExit();
  };

  const certAuthorityName = generateCertAuthorityDisplayName(
    certAuthorityType,
    certAuthority
  );

  return (
    <Modal
      className={baseClass}
      title="Delete certificate authority (CA)"
      onExit={onExit}
    >
      <>
        <p>
          Fleet won&apos;t remove certificates from the certificate authority (
          <b>{certAuthorityName}</b>) on existing hosts.
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
      </>
    </Modal>
  );
};

export default DeleteCertificateAuthorityModal;
