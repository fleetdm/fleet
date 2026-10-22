import React, { useState } from "react";
import { useQuery } from "react-query";

import DataError from "components/DataError";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";
import { IFormErrors } from "hooks/useFormValidation";
import {
  ICertificateAuthorityPartial,
  isEJBCACertAuthority,
} from "interfaces/certificates";
import certificatesAPI from "services/entities/certificates";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import CustomESTForm from "../CustomESTForm";
import CustomSCEPForm from "../CustomSCEPForm";
import DigicertForm from "../DigicertForm";
import EJBCAForm from "../EJBCAForm";
import { IEJBCAFormData } from "../EJBCAForm/EJBCAForm";
import {
  getEJBCAErrorMessage,
  splitEJBCAServerErrors,
} from "../EJBCAForm/helpers";
import HydrantForm from "../HydrantForm";
import NDESForm from "../NDESForm";
import SmallstepForm from "../SmallstepForm";

import {
  generateDefaultFormData,
  generateEditCertAuthorityData,
  getErrorMessage,
  updateFormData,
} from "./helpers";

const baseClass = "edit-cert-authority-modal";

interface IEditCertAuthorityModalProps {
  certAuthority: ICertificateAuthorityPartial;
  /** All the CAs, which an EJBCA CA can't share its name with */
  certAuthorities?: ICertificateAuthorityPartial[];
  onExit: () => void;
}

const EditCertAuthorityModal = ({
  certAuthority,
  certAuthorities,
  onExit,
}: IEditCertAuthorityModalProps) => {
  const [isUpdating, setIsUpdating] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [formData, setFormData] = useState<ICertFormData | undefined>();
  const [
    ejbcaServerErrors,
    setEJBCAServerErrors,
  ] = useState<IFormErrors | null>(null);

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
      notify.success("Successfully edited certificate authority.");
      onExit();
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
    }
    setIsUpdating(false);
  };

  const onEditEJBCA = async (submittedFormData: IEJBCAFormData) => {
    if (!fullCertAuthority) {
      return;
    }
    const editPatchData = generateEditCertAuthorityData(
      fullCertAuthority,
      submittedFormData
    );
    if (
      !("ejbca" in editPatchData) ||
      !Object.keys(editPatchData.ejbca).length
    ) {
      // Nothing was changed, and the server rejects an empty update.
      onExit();
      return;
    }
    setIsUpdating(true);
    try {
      await certificatesAPI.editCertificateAuthority(
        certAuthority.id,
        editPatchData
      );
      notify.success("Successfully edited certificate authority.");
      onExit();
    } catch (e) {
      // The form shows the errors of its fields, and the rest are a toast.
      const { fieldErrors, otherReasons } = splitEJBCAServerErrors(e);
      setEJBCAServerErrors(fieldErrors);
      if (otherReasons.length || Object.keys(fieldErrors).length === 0) {
        notify.error(getEJBCAErrorMessage("edit", otherReasons), {
          response: e,
        });
      }
    }
    setIsUpdating(false);
  };

  const getFormComponent = () => {
    switch (certAuthority.type) {
      case "ndes_scep_proxy":
        return NDESForm;
      case "digicert":
        return DigicertForm;
      case "hydrant":
        return HydrantForm;
      case "smallstep":
        return SmallstepForm;
      case "custom_scep_proxy":
        return CustomSCEPForm;
      case "custom_est_proxy":
        return CustomESTForm;
      default:
        throw new Error(
          `Unknown certificate authority type: ${certAuthority.type}`
        );
    }
  };

  const renderForm = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError className={`${baseClass}__data-error`} />;
    }

    if (certAuthority.type === "ejbca") {
      if (
        !fullCertAuthority ||
        !formData ||
        !isEJBCACertAuthority(fullCertAuthority)
      ) {
        return <></>;
      }
      return (
        <EJBCAForm
          certAuthorities={certAuthorities}
          initialFormData={formData as IEJBCAFormData}
          editingCertAuthorityId={certAuthority.id}
          clientCertExpiresAt={fullCertAuthority.mtls_cert_expires_at}
          submitBtnText="Save"
          isSubmitting={isUpdating}
          serverErrors={ejbcaServerErrors}
          onSubmit={onEditEJBCA}
          onCancel={onExit}
        />
      );
    }

    const FormComponent = getFormComponent();
    if (!FormComponent || !formData) return <></>;

    return (
      <FormComponent
        // @ts-ignore TODO: figure out how to fix this type issue
        formData={formData}
        submitBtnText="Save"
        isSubmitting={isUpdating}
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
