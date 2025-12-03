import React, { useMemo } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { generateFormValidations, validateFormData } from "./helpers";

const baseClass = "okta-form";

export interface IOktaFormData {
  name: string;
  scepURL: string;
  challengeURL: string;
  username: string;
  password: string;
}

interface IOktaFormProps {
  certAuthorities?: ICertificateAuthorityPartial[];
  formData: IOktaFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  isDirty?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const OktaForm = ({
  certAuthorities,
  formData,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  isDirty = true,
  onChange,
  onSubmit,
  onCancel,
}: IOktaFormProps) => {
  const validationsConfig = useMemo(() => {
    return generateFormValidations(certAuthorities ?? [], isEditing);
  }, [certAuthorities, isEditing]);

  const validations = useMemo(() => {
    return validateFormData(formData, validationsConfig);
  }, [formData, validationsConfig]);

  const { name, scepURL, challengeURL, username, password } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  return (
    <form onSubmit={onSubmitForm}>
      <InputField
        label="Name"
        name="name"
        value={name}
        error={validations.name?.message}
        onChange={onChange}
        parseTarget
        placeholder="WIFI_CERTIFICATE"
        helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_OKTA_DATA_WIFI_CERTIFICATE)."
      />
      <InputField
        label="SCEP URL"
        name="scepURL"
        value={scepURL}
        error={validations.scepURL?.message}
        onChange={onChange}
        parseTarget
        placeholder="https://example.okta.com/scep"
      />
      <InputField
        label="Challenge URL"
        name="challengeURL"
        value={challengeURL}
        error={validations.challengeURL?.message}
        onChange={onChange}
        parseTarget
        placeholder="https://example.okta.com/api/v1/scep/challenge"
        helpText="Okta SCEP challenge URL for retrieving dynamic challenge passwords."
      />
      <InputField
        label="Username"
        name="username"
        value={username}
        error={validations.username?.message}
        onChange={onChange}
        parseTarget
        placeholder="admin@example.com"
        helpText="HTTP Basic Authentication username for challenge URL."
      />
      <InputField
        type="password"
        label="Password"
        name="password"
        value={password}
        error={validations.password?.message}
        onChange={onChange}
        parseTarget
        helpText="HTTP Basic Authentication password for challenge URL."
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
            type="submit"
            isLoading={isSubmitting}
            disabled={!validations.isValid || isSubmitting || !isDirty}
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

export default OktaForm;
