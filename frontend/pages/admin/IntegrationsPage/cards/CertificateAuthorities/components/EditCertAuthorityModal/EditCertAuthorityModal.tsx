import React, { useContext, useRef, useState } from "react";

import { NotificationContext } from "context/notification";
import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  isCustomSCEPCertIntegration,
  isDigicertCertIntegration,
  isNDESCertIntegration,
} from "interfaces/integration";

import Modal from "components/Modal";

import DigicertForm from "../DigicertForm";
import {
  generateDefaultFormData,
  generateErrorMessage,
  getCertificateAuthorityType,
} from "./helpers";
import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { get } from "lodash";

const baseClass = "edit-cert-authority-modal";

interface IEditCertAuthorityModalProps {
  listItemId: string;
  certAuthority: ICertificateIntegration;
  onExit: () => void;
}

const EditCertAuthorityModal = ({
  listItemId,
  certAuthority,
  onExit,
}: IEditCertAuthorityModalProps) => {
  const certType = useRef(getCertificateAuthorityType(certAuthority));

  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<ICertFormData | null>(() =>
    generateDefaultFormData(certAuthority)
  );

  const onChangeForm = (update: { name: string; value: string }) => {
    setFormData((prevFormData) => {
      if (!prevFormData) return prevFormData;

      return {
        ...prevFormData,
        [update.name]: update.value,
      };
    });
  };

  const onAddCertAuthority = async () => {
    setIsUpdating(true);
    try {
      // const newConfig = await certificatesAPI.addCertificateAuthority(
      //   addPatchData
      // );
      renderFlash("success", "Successfully added your certificate authority.");
      onExit();
      // setConfig(newConfig);
    } catch (e) {
      renderFlash("error", generateErrorMessage(e));
    }
    setIsUpdating(false);
  };

  const getFormComponent = () => {
    if (isDigicertCertIntegration(certAuthority)) {
      return DigicertForm;
    }
    return null;
  };

  const renderForm = () => {
    const FormComponent = getFormComponent();
    if (!FormComponent || !formData) return <></>;

    return (
      <FormComponent
        formData={formData}
        submitBtnText="Save"
        isSubmitting={isUpdating}
        onChange={onChangeForm}
        onSubmit={onAddCertAuthority}
        onCancel={onExit}
      />
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate authority (CA)"
      width="large"
      onExit={onExit}
    >
      {renderForm()}
    </Modal>
  );
};

export default EditCertAuthorityModal;
