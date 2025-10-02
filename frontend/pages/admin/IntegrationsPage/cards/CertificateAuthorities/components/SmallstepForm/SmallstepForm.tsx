import React, { useMemo } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { generateFormValidations, validateFormData } from "./helpers";

const baseClass = "smallstep-form";

export interface ISmallstepFormData {
  name: string;
  scepURL: string;
  challengeURL: string;
  username: string;
  password: string;
}

interface ISmallstepFormProps {
  certAuthorities?: ICertificateAuthorityPartial[];
  formData: ISmallstepFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  isDirty?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const SmallstepForm = ({
  certAuthorities,
  formData,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  isDirty = true,
  onChange,
  onSubmit,
  onCancel,
}: ISmallstepFormProps) => {
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
      <div className={`${baseClass}__fields`}>
        <InputField
          label="Name"
          name="name"
          value={name}
          error={validations.name?.message}
          onChange={onChange}
          parseTarget
          placeholder="WIFI_CERTIFICATE"
          helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_SMALLSTEP_DATA_WIFI_CERTIFICATE)."
        />
        <InputField
          label="SCEP URL"
          name="scepURL"
          value={scepURL}
          error={validations.scepURL?.message}
          onChange={onChange}
          parseTarget
          placeholder="https://example.scep.smallstep.com/p/agents/integration-fleet-xr9f4db7"
        />
        <InputField
          label="Challenge URL"
          name="challengeURL"
          value={challengeURL}
          error={validations.challengeURL?.message}
          onChange={onChange}
          parseTarget
          placeholder="https://example.scep.smallstep.com/fleet/xr9f4db7-83f1-48ab-8982-8b6870d4fl85/challenge"
          helpText={
            <>
              Smallstep calls this the <b>SCEP Challenge URL</b>.
            </>
          }
        />
        <InputField
          label="Username"
          name="username"
          value={username}
          error={validations.username?.message}
          onChange={onChange}
          parseTarget
          placeholder={"r9c5faea-af93-4679-922c-5548c6254438"}
          helpText={
            <>
              Smallstep calls this the{" "}
              <b>Challenge Basic Authentication Username</b>.
            </>
          }
        />
        <InputField
          type="password"
          label="Password"
          name="password"
          value={password}
          error={validations.password?.message}
          onChange={onChange}
          parseTarget
          helpText={
            <>
              Smallstep calls this the{" "}
              <b>Challenge Basic Authentication Password</b>.
            </>
          }
        />
      </div>
      <div className={`${baseClass}__cta`}>
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

export default SmallstepForm;
