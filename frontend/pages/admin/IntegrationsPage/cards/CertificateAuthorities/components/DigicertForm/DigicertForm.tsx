import React from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "digicert-form";

export interface IDigicertFormData {
  name: string;
  url: string;
  apiToken: string;
  profileId: string;
  commonName: string;
  userPrincipalNames: string;
  certificateSeatId: string;
}

interface IDigicertFormProps {
  submitBtnText: string;
  onSubmit: () => void;
  onCancel: () => void;
}

const DigicertForm = ({
  submitBtnText,
  onSubmit,
  onCancel,
}: IDigicertFormProps) => {
  const isUpdating = false;

  return (
    <form className={baseClass} onSubmit={onSubmit}>
      <div className={`${baseClass}__fields`}>
        <InputField
          name="name"
          label="Name"
          helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_CERT_DATA_DIGICERT_WIFI)."
          placeholder="DIGICERT_WIFI"
        />
        <InputField
          name="url"
          label="URL"
          helpText="DigiCert ONE instance URL."
          value={"https://one.digicert.com"}
        />
        <InputField
          name="apiToken"
          label="API token"
          helpText="DigiCert One API token for service user."
        />
        <InputField
          name="profileId"
          label="Profile GUID"
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
          helpText="Certificates delivered to your hosts will have this CN in the subject."
          placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        />
        <InputField
          name="userPrincipleNames"
          label="User principal name (UPN)"
          helpText="Certificates delivered to your hosts will have this UPN attribute in Subject Alternative Name (SAN)."
          placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        />
        <InputField
          name="certificateSeatId"
          label="Certificate seat ID"
          helpText="Certificates delivered to your hosts will be assigned to this seat ID in DigiCert."
          placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        />
      </div>
      <div className={`${baseClass}__cta`}>
        <TooltipWrapper
          tipContent="Complete all fields to save."
          underline={false}
          position="top"
          showArrow
          // TODO: disable when form is invalid
        >
          <Button isLoading={isUpdating} disabled={isUpdating}>
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
