import React, { useState } from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import { validateFormData, IDigicertFormValidation } from "./helpers";

const baseClass = "digicert-form";

export interface IDigicertFormData {
  name: string;
  url: string;
  apiToken: string;
  profileId: string;
  commonName: string;
  userPrincipalName: string;
  certificateSeatId: string;
}

interface IDigicertFormProps {
  formData: IDigicertFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const DigicertForm = ({
  formData,
  submitBtnText,
  isSubmitting,
  onChange,
  onSubmit,
  onCancel,
}: IDigicertFormProps) => {
  const [formValidation, setFormValidation] = useState<IDigicertFormValidation>(
    {
      isValid: false,
    }
  );

  const {
    name,
    url,
    apiToken,
    profileId,
    commonName,
    userPrincipalName,
    certificateSeatId,
  } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  const onInputChange = (update: { name: string; value: string }) => {
    setFormValidation(
      validateFormData({ ...formData, [update.name]: update.value })
    );
    onChange(update);
  };

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <div className={`${baseClass}__fields`}>
        <InputField
          name="name"
          label="Name"
          value={name}
          onChange={onInputChange}
          error={formValidation.name?.message}
          helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_CERT_DATA_DIGICERT_WIFI)."
          parseTarget
          placeholder="DIGICERT_WIFI"
        />
        <InputField
          name="url"
          label="URL"
          value={url}
          onChange={onInputChange}
          error={formValidation.url?.message}
          parseTarget
          helpText="DigiCert ONE instance URL."
        />
        <InputField
          type="password"
          name="apiToken"
          label="API token"
          value={apiToken}
          onChange={onInputChange}
          parseTarget
          helpText="DigiCert One API token for service user."
        />
        <InputField
          name="profileId"
          label="Profile GUID"
          value={profileId}
          onChange={onInputChange}
          parseTarget
          helpText={
            <>
              You can find the <b>Profile GUID</b> by opening one of the{" "}
              <CustomLink
                text="Digicert profiles"
                url="https://demo.one.digicert.com/mpki/policies/profiles"
                newTab
              />
            </>
          }
        />
        <InputField
          name="commonName"
          label="Certificate common name (CN)"
          value={commonName}
          onChange={onInputChange}
          parseTarget
          helpText="Certificates delivered to your hosts will have this CN in the subject."
          placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        />
        <InputField
          name="userPrincipalName"
          label="User principal name (UPN)"
          value={userPrincipalName}
          onChange={onInputChange}
          parseTarget
          helpText="Certificates delivered to your hosts will have this UPN attribute in Subject Alternative Name (SAN). (optional)"
          placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        />
        <InputField
          name="certificateSeatId"
          label="Certificate seat ID"
          value={certificateSeatId}
          onChange={onInputChange}
          parseTarget
          helpText="Certificates delivered to your hosts will be assigned to this seat ID in DigiCert."
          placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        />
      </div>
      <div className={`${baseClass}__cta`}>
        <TooltipWrapper
          tipContent="Complete all required fields to save."
          underline={false}
          position="top"
          disableTooltip={formValidation.isValid}
          showArrow
        >
          <Button
            isLoading={isSubmitting}
            disabled={!formValidation.isValid || isSubmitting}
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

export default DigicertForm;
