import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import certificatesAPI from "services/entities/certificates";

import Modal from "components/Modal";
import DigicertForm from "../DigicertForm";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { useCertAuthorityDataGenerator } from "../DeleteCertificateAuthorityModal/helpers";
import { generateErrorMessage } from "./helpers";

const baseClass = "add-cert-authority-modal";

interface IAddCertAuthorityModalProps {
  onExit: () => void;
}

const AddCertAuthorityModal = ({ onExit }: IAddCertAuthorityModalProps) => {
  const { generateAddPatchData } = useCertAuthorityDataGenerator("digicert");

  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<IDigicertFormData>({
    name: "",
    url: "https://one.digicert.com",
    apiToken: "",
    profileId: "",
    commonName: "",
    userPrincipalName: "",
    certificateSeatId: "",
  });

  const onChange = (update: { name: string; value: string }) => {
    setFormData({ ...formData, [update.name]: update.value });
  };

  const onAddCertAuthority = async () => {
    const addPatchData = generateAddPatchData(formData);
    console.log(addPatchData);

    setIsUpdating(true);
    try {
      await certificatesAPI.addCertificateAuthority(addPatchData);
      renderFlash("success", "Successfully added your certificate authority.");
    } catch (e) {
      renderFlash("error", generateErrorMessage(e));
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
      <DigicertForm
        formData={formData}
        submitBtnText="Add CA"
        onChange={onChange}
        onSubmit={onAddCertAuthority}
        onCancel={onExit}
      />
    </Modal>
  );
};

export default AddCertAuthorityModal;
