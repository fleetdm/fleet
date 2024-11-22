import React, { useCallback, useState } from "react";

import { IFormField } from "interfaces/form_field";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";

import { IPkiConfig, IPkiTemplate } from "interfaces/pki";

const baseClass = "pki-edit-template-modal";

type IFormErrors = Partial<Record<keyof IPkiTemplate, string>>;

const TEMPLATE_PLACEHOLDERS: Record<keyof IPkiTemplate, string> = {
  profile_id: "123",
  name: "DIGICERT_TEMPLATE",
  common_name: "$FLEET_VAR_HOST_HARDWARE_SERIAL@example.com",
  san: "$FLEET_VAR_HOST_HARDWARE_SERIAL@example.com",
  seat_id: "$FLEET_VAR_HOST_HARDWARE_SERIAL@example.com",
};

const TEMPLATE_HELP_TEXT: Record<
  keyof IPkiTemplate,
  string | React.ReactNode
> = {
  profile_id: (
    <span>
      The <b>Certificate profile ID</b> field in DigiCert.
    </span>
  ),
  name:
    "Letters, numbers, and underscores only. Fleet will create a configuration profile variable with the $FLEET_VAR_PKI_CERT_ prefix (e.g. $FLEET_VAR_PKI_CERT_DIGICERT_TEMPALTE).",
  common_name: "Certificates delivered to your hosts using will have this CN.",
  san: "Certificates delivered to your hosts using will have this SAN.",
  seat_id:
    "Certificates delivered to your hosts using will be assgined to this seat ID in DigiCert.",
};

const EditTemplateModal = ({
  pkiConfig,
  onCancel,
  onSuccess,
}: {
  pkiConfig: IPkiConfig;
  onCancel: () => void;
  onSuccess: () => void;
}) => {
  const [formData, setFormData] = useState<IPkiTemplate>(
    pkiConfig.templates[0] || {
      profile_id: "",
      name: "",
      common_name: "",
      san: "",
      seat_id: "",
    }
  );
  const [formErrors, setFormErrors] = useState<IFormErrors>({});

  const onInputChange = ({ name, value }: IFormField) => {
    setFormErrors((prev) => ({ ...prev, [name]: "" }));
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const onSubmit = useCallback(async () => {
    // validate
    const errors: IFormErrors = {};

    if (!formData.profile_id) {
      errors.profile_id = "Profile ID is required";
    }

    if (!formData.name) {
      errors.name = "Name is required";
    }

    if (!formData.common_name) {
      errors.common_name = "Common name is required";
    }

    if (!formData.san) {
      errors.san = "Subject alternative name is required";
    }

    if (!formData.seat_id) {
      errors.seat_id = "Seat ID is required";
    }

    if (Object.keys(errors).length) {
      setFormErrors(errors);
      return;
    }

    // save
    console.log("Save", formData);
    onSuccess();
  }, [formData, onSuccess]);

  const disableInput = !pkiConfig.templates.length;
  const disableSave = Object.values(formData).some((v) => !v);

  const isSaving = false;

  return (
    <Modal
      title={disableInput ? "Certificate template" : "Add template"}
      onExit={onCancel}
    >
      <>
        <form onSubmit={onSubmit} autoComplete="off">
          <InputField
            inputWrapperClass={`${baseClass}__admin-url-input`}
            label="Name"
            name="name"
            value={formData.name}
            onChange={onInputChange}
            parseTarget
            error={formErrors.name}
            placeholder={TEMPLATE_PLACEHOLDERS.name}
            helpText={TEMPLATE_HELP_TEXT.name}
            disabled={disableInput}
          />
          <InputField
            inputWrapperClass={`${baseClass}__scep-url-input`}
            label="Profile ID"
            name="profile_id"
            value={formData.profile_id}
            onChange={onInputChange}
            parseTarget
            error={formErrors.profile_id}
            placeholder={TEMPLATE_PLACEHOLDERS.profile_id.toString()}
            helpText={TEMPLATE_HELP_TEXT.profile_id}
            disabled={disableInput}
          />
          <InputField
            inputWrapperClass={`${baseClass}__admin-url-input`}
            label="Certificate common name (CN)"
            name="coomon_name"
            value={formData.common_name}
            onChange={onInputChange}
            parseTarget
            error={formErrors.common_name}
            placeholder={TEMPLATE_PLACEHOLDERS.common_name}
            helpText={TEMPLATE_HELP_TEXT.common_name}
            disabled={disableInput}
          />
          <InputField
            inputWrapperClass={`${baseClass}__username-input`}
            label="Certificate subject alternative name (SAN)"
            name="san"
            value={formData.san}
            onChange={onInputChange}
            parseTarget
            placeholder={TEMPLATE_PLACEHOLDERS.san}
            error={formErrors.san}
            helpText={TEMPLATE_HELP_TEXT.san}
            disabled={disableInput}
          />
          <InputField
            inputWrapperClass={`${baseClass}__password-input`}
            label="Certificate seat ID"
            name="seat_id"
            value={formData.seat_id}
            onChange={onInputChange}
            parseTarget
            placeholder={TEMPLATE_PLACEHOLDERS.seat_id}
            error={formErrors.seat_id}
            helpText={TEMPLATE_HELP_TEXT.seat_id}
            disabled={disableInput}
          />
          <div className="modal-cta-wrap">
            <TooltipWrapper
              tipContent="Complete all fields to save"
              position="top"
              showArrow
              underline={false}
              tipOffset={8}
              disableTooltip={!disableSave || isSaving}
            >
              <Button
                variant="brand"
                type="submit"
                isLoading={isSaving}
                disabled={disableSave}
              >
                Save
              </Button>
            </TooltipWrapper>
            <Button variant="inverse" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </form>
      </>
    </Modal>
  );
};

export default EditTemplateModal;
