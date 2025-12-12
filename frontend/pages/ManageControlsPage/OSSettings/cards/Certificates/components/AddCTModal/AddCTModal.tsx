import React, { useContext, useEffect, useMemo, useState } from "react";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import certificatesAPI, { ICertTemplate } from "services/entities/certificates";
import { ICertificateAuthorityPartial } from "interfaces/certificates";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";

import {
  validateFormData,
  generateFormValidations,
  IAddCTFormData,
  IAddCTFormValidation,
} from "./helpers";

const baseClass = "add-ct-modal";

interface IAddCTModalProps {
  existingCTs: ICertTemplate[];
  onExit: () => void;
  onSuccess?: () => void;
}

const AddCTModal = ({ existingCTs, onExit, onSuccess }: IAddCTModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { currentTeam } = useContext(AppContext);

  const [isCreating, setIsCreating] = useState(false);
  const [formData, setFormData] = useState<IAddCTFormData>({
    name: "",
    certAuthorityId: "",
    subjectName: "",
  });

  const validations = useMemo(
    () => generateFormValidations(existingCTs || []),
    [existingCTs]
  );

  const [formValidation, setFormValidation] = useState<IAddCTFormValidation>(
    () => validateFormData(formData, validations)
  );

  // Fetch certificate authorities
  const {
    data: certAuthoritiesResp,
    isLoading: isLoadingCAs,
    isError: isErrorCAs,
  } = useQuery<{ certificate_authorities: ICertificateAuthorityPartial[] }>(
    ["certificate_authorities"],
    () => certificatesAPI.getCertificateAuthoritiesList(),
    {
      retry: false,
      refetchOnWindowFocus: false,
    }
  );

  const certAuthorities = certAuthoritiesResp?.certificate_authorities || [];

  // Generate dropdown options from certificate authorities
  const caDropdownOptions = useMemo(() => {
    return certAuthorities.map((ca) => ({
      value: ca.id.toString(),
      label: ca.name,
    }));
  }, [certAuthorities]);

  const onInputChange = (update: { name: string; value: string }) => {
    const updatedFormData = { ...formData, [update.name]: update.value };
    setFormData(updatedFormData);
    setFormValidation(validateFormData(updatedFormData, validations));
  };

  const onDropdownChange = (value: string) => {
    const updatedFormData = { ...formData, certAuthorityId: value };
    setFormData(updatedFormData);
    setFormValidation(validateFormData(updatedFormData, validations));
  };

  const onSubmitForm = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    setIsCreating(true);
    try {
      await certificatesAPI.createCertTemplate({
        name: formData.name,
        certAuthorityId: parseInt(formData.certAuthorityId, 10),
        subjectName: formData.subjectName,
        teamId: currentTeam?.id,
      });
      renderFlash("success", "Successfully added your certificate template.");
      if (onSuccess) {
        onSuccess();
      }
      onExit();
    } catch (e) {
      renderFlash(
        "error",
        "Couldn't add certificate template. Please try again."
      );
    } finally {
      setIsCreating(false);
    }
  };

  const renderForm = () => {
    if (isLoadingCAs) {
      return <div>Loading certificate authorities...</div>;
    }

    if (isErrorCAs || certAuthorities.length === 0) {
      return (
        <div>
          No certificate authorities available. Please add a certificate
          authority first.
        </div>
      );
    }

    return (
      <form className={baseClass} onSubmit={onSubmitForm}>
        <InputField
          name="name"
          label="Name"
          value={formData.name}
          onChange={onInputChange}
          error={formValidation.name?.message}
          helpText="Letters, numbers, spaces, dashes, and underscores only. Name can be used as certificate alias to reference in configuration profiles."
          parseTarget
          placeholder="VPN certificate"
        />
        <Dropdown
          label="Certificate authority (CA)"
          options={caDropdownOptions}
          value={formData.certAuthorityId}
          onChange={onDropdownChange}
          placeholder="Select certificate authority"
          helpText="Certificate will be issued from this CA. Currently, only custom SCEP CA is supported. You can add CAs on the Certificate authorities page."
          searchable={false}
          error={formValidation.certAuthorityId?.message}
        />
        <InputField
          name="subjectName"
          label="Subject name (SN)"
          type="textarea"
          value={formData.subjectName}
          onChange={onInputChange}
          error={formValidation.subjectName?.message}
          helpText='Separate subject fields by a "/". For example: /CN=john@example.com/O=Acme Inc.'
          parseTarget
          placeholder="/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/O=Your Organization"
        />
        <div className="modal-cta-wrap">
          <TooltipWrapper
            tipContent="Complete all required fields to save"
            underline={false}
            position="top"
            disableTooltip={formValidation.isValid}
            showArrow
          >
            <Button
              isLoading={isCreating}
              disabled={!formValidation.isValid || isCreating}
              type="submit"
            >
              Create
            </Button>
          </TooltipWrapper>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </form>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate template"
      width="large"
      onExit={onExit}
      isContentDisabled={isCreating}
    >
      {renderForm()}
    </Modal>
  );
};

export default AddCTModal;
