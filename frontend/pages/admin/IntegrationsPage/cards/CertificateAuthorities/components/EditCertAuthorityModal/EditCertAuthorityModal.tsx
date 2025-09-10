import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { NotificationContext } from "context/notification";
import { ICertificateAuthorityPartial } from "interfaces/certificates";
import certificatesAPI from "services/entities/certificates";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import {
  generateDefaultFormData,
  generateEditCertAuthorityData,
  getErrorMessage,
  updateFormData,
} from "./helpers";

import DigicertForm from "../DigicertForm";
import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import NDESForm from "../NDESForm";
import CustomSCEPForm from "../CustomSCEPForm";
import HydrantForm from "../HydrantForm";

const baseClass = "edit-cert-authority-modal";

interface IEditCertAuthorityModalProps {
  certAuthority: ICertificateAuthorityPartial;
  onExit: () => void;
}

const EditCertAuthorityModal = ({
  certAuthority,
  onExit,
}: IEditCertAuthorityModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [formData, setFormData] = useState<ICertFormData | undefined>();

  const { data: fullCertAuthority, isLoading, isError } = useQuery(
    ["cert-authority", certAuthority.id],
    () => certificatesAPI.getCertificateAuthority(certAuthority.id),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      onSuccess: (data) => {
        setFormData(generateDefaultFormData(data));
      },
    }
  );

  const onChangeForm = (update: { name: string; value: string }) => {
    if (!fullCertAuthority) return;
    setFormData((prevFormData) => {
      if (!prevFormData) return prevFormData;
      return updateFormData(fullCertAuthority, prevFormData, update);
    });

    setIsDirty(true);
  };

  const onEditCertAuthority = async () => {
    if (!fullCertAuthority || !formData) {
      return;
    }
    const editPatchData = generateEditCertAuthorityData(
      fullCertAuthority,
      formData
    );
    setIsUpdating(true);
    try {
      await certificatesAPI.editCertificateAuthority(
        certAuthority.id,
        editPatchData
      );
      renderFlash("success", "Successfully edited your certificate authority.");
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
    setIsUpdating(false);
  };

  const getFormComponent = () => {
    if (certAuthority.type === "ndes_scep_proxy") {
      return NDESForm;
    }
    if (certAuthority.type === "digicert") {
      return DigicertForm;
    }
    if (certAuthority.type === "hydrant") {
      return HydrantForm;
    }
    return CustomSCEPForm;
  };

  const renderForm = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError className={`${baseClass}__data-error`} />;
    }

    const FormComponent = getFormComponent();
    if (!FormComponent || !formData) return <></>;

    return (
      <FormComponent
        // @ts-ignore TODO: figure out how to fix this type issue
        formData={formData}
        submitBtnText="Save"
        isSubmitting={isUpdating}
        isEditing
        isDirty={isDirty}
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
