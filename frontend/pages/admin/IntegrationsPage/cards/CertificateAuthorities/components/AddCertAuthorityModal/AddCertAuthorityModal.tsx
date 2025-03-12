import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import certificatesAPI from "services/entities/certificates";
import { ICertificateAuthorityType } from "interfaces/integration";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";

import { generateErrorMessage } from "./helpers";
import DigicertForm from "../DigicertForm";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { useCertAuthorityDataGenerator } from "../DeleteCertificateAuthorityModal/helpers";

const baseClass = "add-cert-authority-modal";

interface IAddCertAuthorityModalProps {
  onExit: () => void;
}

const AddCertAuthorityModal = ({ onExit }: IAddCertAuthorityModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [
    certAuthorityType,
    setCertAuthorityType,
  ] = useState<ICertificateAuthorityType>("digicert");
  const [isAdding, setIsAdding] = useState(false);
  const [formData, setFormData] = useState<IDigicertFormData>({
    name: "",
    url: "https://one.digicert.com",
    apiToken: "",
    profileId: "",
    commonName: "",
    userPrincipalName: "",
    certificateSeatId: "",
  });
  const { generateAddPatchData } = useCertAuthorityDataGenerator(
    certAuthorityType
  );

  const onChangeDropdown = (value: ICertificateAuthorityType) => {
    setCertAuthorityType(value);
  };

  const onChangeForm = (update: { name: string; value: string }) => {
    setFormData({ ...formData, [update.name]: update.value });
  };

  const onAddCertAuthority = async () => {
    const addPatchData = generateAddPatchData(formData);
    setIsAdding(true);
    try {
      await certificatesAPI.addCertificateAuthority(addPatchData);
      renderFlash("success", "Successfully added your certificate authority.");
    } catch (e) {
      renderFlash("error", generateErrorMessage(e));
    }
    onExit();
    setIsAdding(false);
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate authority (CA)"
      width="large"
      onExit={onExit}
    >
      <>
        <Dropdown
          options={[{ label: "Digicert", value: "digicert" }]}
          value={certAuthorityType}
          className={`${baseClass}__cert-authority-dropdown`}
          onChange={onChangeDropdown}
          searchable={false}
        />
        <DigicertForm
          formData={formData}
          submitBtnText="Add CA"
          isSubmitting={isAdding}
          onChange={onChangeForm}
          onSubmit={onAddCertAuthority}
          onCancel={onExit}
        />
      </>
    </Modal>
  );
};

export default AddCertAuthorityModal;
