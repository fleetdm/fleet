import React, { useCallback, useState } from "react";

import { NotificationContext } from "context/notification";

import { IFormField } from "interfaces/form_field";
import { IPkiConfig, IPkiTemplate } from "interfaces/pki";

import pkiApi from "services/entities/pki";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";

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

// TODO: we should revisit this in design after the PoC
const flattenTemplate = (template: IPkiTemplate | undefined) => {
  if (!template) {
    return {
      profile_id: "",
      name: "",
      common_name: "",
      san: "",
      seat_id: "",
    };
  }
  return {
    profile_id: template.profile_id.toString(),
    name: template.name,
    common_name: template.common_name,
    san: template.san.user_principal_names[0],
    seat_id: template.seat_id,
  };
};

// TODO: we should revisit this in design after the PoC
const unflattenTemplate = (formData: Record<keyof IPkiTemplate, string>) => {
  return {
    profile_id: formData.profile_id,
    name: formData.name,
    common_name: formData.common_name,
    san: { user_principal_names: [formData.san] },
    seat_id: formData.seat_id,
  };
};

const EditTemplateModal = ({
  selectedConfig,
  byPkiName,
  onCancel,
  onSuccess,
}: {
  selectedConfig: IPkiConfig;
  byPkiName: Record<string, IPkiConfig>;
  onCancel: () => void;
  onSuccess: () => void;
}) => {
  const { renderFlash } = React.useContext(NotificationContext);

  const [formData, setFormData] = useState<Record<keyof IPkiTemplate, string>>(
    flattenTemplate(selectedConfig.templates[0])
  );
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [isSaving, setIsSaving] = useState(false);

  const onInputChange = ({ name, value }: IFormField) => {
    setFormErrors((prev) => ({ ...prev, [name]: "" }));
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const onSubmit = useCallback(
    async (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();
      // TODO: validations

      setIsSaving(true);

      // TODO: Need specs for how we handle multiple array elements at top-level (i.e. certs by pki_name) and at cert-level
      // (templates by template name). For now, we're just going to replace all templates for the
      // selected pki_name and preserve existing config data associated to any other pki_name.

      const payload: IPkiConfig[] = [];
      Object.entries(byPkiName).forEach(([pkiName, config]) => {
        if (pkiName === selectedConfig.pki_name) {
          payload.push({
            pki_name: pkiName,
            templates: [unflattenTemplate(formData)],
          });
        } else {
          payload.push(config);
        }
      });

      try {
        await pkiApi.patchIntegrationsDigicertPki(payload);
        renderFlash(
          "success",
          <>
            Successfully added certificate template for your{" "}
            <b>{selectedConfig.pki_name}</b> PKI
          </>
        );
        onSuccess();
      } catch {
        renderFlash("error", "Could not save template");
      } finally {
        setIsSaving(false);
      }
    },
    [formData, onSuccess, selectedConfig.pki_name, byPkiName, renderFlash]
  );

  const disableInput = !!selectedConfig.templates.length;
  const disableSave = Object.values(formData).some((v) => !v);

  return (
    <Modal
      title={disableInput ? "Certificate template" : "Add template"}
      onExit={onCancel}
    >
      <>
        <form onSubmit={onSubmit} autoComplete="off">
          <InputField
            inputWrapperClass={`${baseClass} ${baseClass}__admin-url-input`}
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
            inputWrapperClass={`${baseClass} ${baseClass}__scep-url-input`}
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
            inputWrapperClass={`${baseClass} ${baseClass}__admin-url-input`}
            label="Certificate common name (CN)"
            name="common_name"
            value={formData.common_name}
            onChange={onInputChange}
            parseTarget
            error={formErrors.common_name}
            placeholder={TEMPLATE_PLACEHOLDERS.common_name}
            helpText={TEMPLATE_HELP_TEXT.common_name}
            disabled={disableInput}
          />
          <InputField
            inputWrapperClass={`${baseClass} ${baseClass}__username-input`}
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
            inputWrapperClass={`${baseClass} ${baseClass}__password-input`}
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
            {disableInput ? (
              <Button variant="brand" type="button" onClick={onCancel}>
                Done
              </Button>
            ) : (
              <>
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
              </>
            )}
          </div>
        </form>
      </>
    </Modal>
  );
};

export default EditTemplateModal;
