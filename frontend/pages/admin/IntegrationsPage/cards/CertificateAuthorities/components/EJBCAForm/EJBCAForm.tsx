import React, { useMemo, useState } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";

import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import TooltipWrapper from "components/TooltipWrapper";
import {
  generateFormValidations,
  IEJBCAFormValidation,
  readFileAsBase64,
  validateFormData,
} from "./helpers";

const baseClass = "ejbca-form";

export interface IEJBCAFormData {
  name: string;
  url: string;
  /** base64-encoded contents of the uploaded .p12 — empty when not rotating */
  clientP12Base64: string;
  /** the original filename, used purely for display */
  clientP12FileName: string;
  clientP12Password: string;
  trustCABundle: string;
  certificateAuthorityNameEJBCA: string;
  certificateProfileName: string;
  endEntityProfileName: string;
  usernameTemplate: string;
  /** single UPN to embed in the SAN otherName extension; the backend accepts
   * an array so we wrap this into [upn] on submit. Empty string = no UPN. */
  userPrincipalName: string;
}

interface IEJBCAFormProps {
  certAuthorities?: ICertificateAuthorityPartial[];
  formData: IEJBCAFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  isDirty?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const EJBCAForm = ({
  certAuthorities,
  formData,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  isDirty = true,
  onChange,
  onSubmit,
  onCancel,
}: IEJBCAFormProps) => {
  const validations = useMemo(
    () => generateFormValidations(certAuthorities ?? [], isEditing),
    [certAuthorities, isEditing]
  );

  const [formValidation, setFormValidation] = useState<IEJBCAFormValidation>(
    () => validateFormData(formData, validations)
  );
  const [p12Error, setP12Error] = useState<string | undefined>();

  const {
    name,
    url,
    clientP12FileName,
    clientP12Password,
    trustCABundle,
    certificateAuthorityNameEJBCA,
    certificateProfileName,
    endEntityProfileName,
    usernameTemplate,
    userPrincipalName,
  } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  const onInputChange = (update: { name: string; value: string }) => {
    setFormValidation(
      validateFormData(
        { ...formData, [update.name]: update.value },
        validations
      )
    );
    onChange(update);
  };

  const onP12Upload = async (files: FileList | null) => {
    setP12Error(undefined);
    const file = files?.[0];
    if (!file) {
      return;
    }
    // The .p12 extension is conventional; we don't enforce it strictly
    // because EJBCA's RA Web can also emit .pfx for the same format. The
    // backend rejects anything that isn't a valid PKCS#12 on decode.
    try {
      const base64 = await readFileAsBase64(file);
      onInputChange({ name: "clientP12Base64", value: base64 });
      onInputChange({ name: "clientP12FileName", value: file.name });
    } catch (err) {
      setP12Error("Couldn't read the file. Please try again.");
    }
  };

  const onP12Delete = () => {
    onInputChange({ name: "clientP12Base64", value: "" });
    onInputChange({ name: "clientP12FileName", value: "" });
    onInputChange({ name: "clientP12Password", value: "" });
  };

  const p12Label = isEditing
    ? "Client certificate (.p12) — upload to rotate"
    : "Client certificate (.p12)";

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <InputField
        name="name"
        label="Name"
        value={name}
        onChange={onInputChange}
        error={formValidation.name?.message}
        helpText="Letters, numbers, and underscores only. Fleet creates configuration profile variables with this name as suffix (e.g. $FLEET_VAR_EJBCA_DATA_WIFI_CERTIFICATE)."
        parseTarget
        placeholder="WIFI_CERTIFICATE"
      />
      <InputField
        name="url"
        label="EJBCA REST URL"
        value={url}
        onChange={onInputChange}
        error={formValidation.url?.message}
        parseTarget
        helpText="Base URL of your EJBCA REST API endpoint."
        placeholder="https://ejbca.example.com:8443"
      />
      <FileUploader
        graphicName="file-pem"
        accept=".p12,.pfx,application/x-pkcs12"
        label={p12Label}
        message={
          clientP12FileName ||
          "Drop your PKCS#12 (.p12) file here or click Upload."
        }
        additionalInfo="Enroll a service certificate in EJBCA (Client Authentication EKU), bind it to a Fleet admin role, then upload the .p12 here."
        buttonMessage="Upload"
        internalError={p12Error || formValidation.clientP12Base64?.message}
        onFileUpload={onP12Upload}
        fileDetails={
          clientP12FileName ? { name: clientP12FileName } : undefined
        }
        onDeleteFile={onP12Delete}
        canEdit
      />
      <InputField
        type="password"
        name="clientP12Password"
        label={
          isEditing
            ? "PKCS#12 password (required only when uploading a new .p12)"
            : "PKCS#12 password"
        }
        value={clientP12Password}
        onChange={onInputChange}
        parseTarget
        helpText="Used once to decrypt the uploaded .p12; never stored."
      />
      <InputField
        type="textarea"
        name="trustCABundle"
        label="Trust CA bundle (PEM) — optional"
        value={trustCABundle}
        onChange={onInputChange}
        parseTarget
        helpText="Paste the EJBCA-issuing CA chain (PEM) only if EJBCA's HTTPS certificate isn't already trusted by Fleet's system root store (typical for self-hosted EJBCA)."
        placeholder="-----BEGIN CERTIFICATE-----..."
      />
      <InputField
        name="certificateAuthorityNameEJBCA"
        label="EJBCA Certificate Authority name"
        value={certificateAuthorityNameEJBCA}
        onChange={onInputChange}
        parseTarget
        helpText="The name of the issuing CA inside EJBCA (free text — must match your EJBCA config exactly)."
        placeholder="WifiIssuingCA"
      />
      <InputField
        name="certificateProfileName"
        label="EJBCA Certificate Profile name"
        value={certificateProfileName}
        onChange={onInputChange}
        parseTarget
        helpText='Must permit "Allow Extension Override" if you use UPN templating.'
        placeholder="WifiClientProfile"
      />
      <InputField
        name="endEntityProfileName"
        label="EJBCA End Entity Profile name"
        value={endEntityProfileName}
        onChange={onInputChange}
        parseTarget
        helpText="Must be configured to permit user-supplied passwords. EJBCA Enterprise also requires auto-create-EE to be enabled for fleet-scale enrollment."
        placeholder="WifiUsers"
      />
      <InputField
        name="usernameTemplate"
        label="Username template"
        value={usernameTemplate}
        onChange={onInputChange}
        parseTarget
        helpText="Used as both the EJBCA end-entity username and the CSR Common Name (CN). Supports $FLEET_VAR_HOST_HARDWARE_SERIAL, $FLEET_VAR_HOST_PLATFORM, $FLEET_VAR_HOST_END_USER_EMAIL_IDP."
        placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
      />
      <InputField
        name="userPrincipalName"
        label="User principal name (UPN) — optional"
        value={userPrincipalName}
        onChange={onInputChange}
        parseTarget
        helpText="Embedded in the certificate's Subject Alternative Name as a Microsoft UPN otherName. Supports the same Fleet variables as the username template."
        placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
      />
      <div className="modal-cta-wrap">
        <TooltipWrapper
          tipContent="Complete all required fields to save."
          underline={false}
          position="top"
          disableTooltip={formValidation.isValid}
          showArrow
        >
          <Button
            isLoading={isSubmitting}
            disabled={!formValidation.isValid || isSubmitting || !isDirty}
            type="submit"
          >
            {submitBtnText}
          </Button>
        </TooltipWrapper>
        <Button variant="inverse" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default EJBCAForm;
