import React, { useContext, useRef, useState } from "react";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import {
  ICertificateIntegration,
  isDigicertCertIntegration,
  isNDESCertIntegration,
} from "interfaces/integration";
import certificatesAPI from "services/entities/certificates";

import Modal from "components/Modal";

import {
  generateDefaultFormData,
  getErrorMessage,
  getCertificateAuthorityType,
  updateFormData,
} from "./helpers";

import DigicertForm from "../DigicertForm";
import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { useCertAuthorityDataGenerator } from "../DeleteCertificateAuthorityModal/helpers";
import NDESForm from "../NDESForm";
import CustomSCEPForm from "../CustomSCEPForm";

const baseClass = "edit-cert-authority-modal";

interface IEditCertAuthorityModalProps {
  certAuthority: ICertificateIntegration;
  onExit: () => void;
}

const EditCertAuthorityModal = ({
  certAuthority,
  onExit,
}: IEditCertAuthorityModalProps) => {
  const certType = useRef(getCertificateAuthorityType(certAuthority));
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<ICertFormData>(() =>
    generateDefaultFormData(certAuthority)
  );
  const { generateEditPatchData } = useCertAuthorityDataGenerator(
    certType.current,
    certAuthority
  );

  const onChangeForm = (update: { name: string; value: string }) => {
    setFormData((prevFormData) => {
      if (!prevFormData) return prevFormData;
      return updateFormData(certAuthority, prevFormData, update);
    });
  };

  const onEditCertAuthority = async () => {
    const editPatchData = generateEditPatchData(formData);
    setIsUpdating(true);
    try {
      const newConfig = await certificatesAPI.editCertAuthorityModal(
        editPatchData
      );
      renderFlash("success", "Successfully edited your certificate authority.");
      onExit();
      setConfig(newConfig);
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
    setIsUpdating(false);
  };

  const getFormComponent = () => {
    if (isNDESCertIntegration(certAuthority)) {
      return NDESForm;
    }
    if (isDigicertCertIntegration(certAuthority)) {
      return DigicertForm;
    }
    return CustomSCEPForm;
  };

  const renderForm = () => {
    const FormComponent = getFormComponent();
    if (!FormComponent || !formData) return <></>;

    return (
      <FormComponent
        // @ts-ignore TODO: figure out how to fix this type issue
        formData={formData}
        submitBtnText="Save"
        isSubmitting={isUpdating}
        isEditing
        onChange={onChangeForm}
        onSubmit={onEditCertAuthority}
        onCancel={onExit}
      />
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Edit certificate authority (CA)"
      width="large"
      onExit={onExit}
      isContentDisabled={isUpdating}
    >
      {renderForm()}
    </Modal>
  );
};

export default EditCertAuthorityModal;
