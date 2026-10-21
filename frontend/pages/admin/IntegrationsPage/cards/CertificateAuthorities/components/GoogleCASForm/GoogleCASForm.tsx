import React, { useMemo } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import { ICertificateAuthorityPartial } from "interfaces/certificates";
import { MAX_ENTITY_CHAR_LENGTH } from "utilities/constants";

import { generateFormValidations, validateFormData } from "./helpers";

const baseClass = "google-cas-form";

export interface IGoogleCASFormData {
  name: string;
  projectId: string;
  caPool: string;
  location: string;
  apiKeyJson: string;
}

interface IGoogleCASFormProps {
  certAuthorities?: ICertificateAuthorityPartial[];
  formData: IGoogleCASFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  isDirty?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const GoogleCASForm = ({
  certAuthorities,
  formData,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  isDirty = true,
  onChange,
  onSubmit,
  onCancel,
}: IGoogleCASFormProps) => {
  const validationsConfig = useMemo(
    () => generateFormValidations(certAuthorities ?? [], isEditing),
    [certAuthorities, isEditing]
  );

  const validations = useMemo(
    () => validateFormData(formData, validationsConfig),
    [formData, validationsConfig]
  );

  const { name, projectId, caPool, location, apiKeyJson } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <InputField
        name="name"
        label="Name"
        value={name}
        onChange={onChange}
        error={validations.name?.message}
        helpText="Letters, numbers, and underscores only."
        parseTarget
        placeholder="WIFI_CERTIFICATE"
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        name="projectId"
        label="Project ID"
        value={projectId}
        onChange={onChange}
        error={validations.projectId?.message}
        helpText="The Google Cloud project ID."
        parseTarget
        placeholder="my-project"
      />
      <InputField
        name="caPool"
        label="CA pool"
        value={caPool}
        onChange={onChange}
        error={validations.caPool?.message}
        helpText={
          <>
            You can find your <b>CA pool</b> in the{" "}
            <CustomLink
              text="CA pool manager"
              url="https://console.cloud.google.com/security/cas/caPools"
              newTab
            />
          </>
        }
        parseTarget
        placeholder="my-ca-pool"
      />
      <InputField
        name="location"
        label="Location"
        value={location}
        onChange={onChange}
        error={validations.location?.message}
        parseTarget
        placeholder="us-central1"
      />
      <InputField
        type="password"
        name="apiKeyJson"
        label="Service account key"
        value={apiKeyJson}
        onChange={onChange}
        error={validations.apiKeyJson?.message}
        blockAutoComplete
        helpText={
          <>
            The service account key with the{" "}
            <b>CA Service Certificate Manager</b> role.{" "}
            <CustomLink
              text="Google Cloud's Service accounts"
              url="https://console.cloud.google.com/iam-admin/serviceaccounts"
              newTab
            />
          </>
        }
        parseTarget
      />
      <div className="modal-cta-wrap">
        <TooltipWrapper
          tipContent="Complete all required fields to save."
          underline={false}
          position="top"
          disableTooltip={validations.isValid}
          showArrow
        >
          <Button
            isLoading={isSubmitting}
            disabled={!validations.isValid || isSubmitting || !isDirty}
            type="submit"
          >
            {submitBtnText}
          </Button>
        </TooltipWrapper>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default GoogleCASForm;
