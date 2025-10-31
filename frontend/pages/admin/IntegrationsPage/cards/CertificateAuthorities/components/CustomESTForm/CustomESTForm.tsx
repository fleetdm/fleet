import React, { useMemo } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { generateFormValidations, validateFormData } from "./helpers";

const baseClass = "custom-e-s-t-form";

export interface ICustomESTFormData {
  name: string;
  url: string;
  username: string;
  password: string;
}
interface ICustomESTFormProps {
  formData: ICustomESTFormData;
  certAuthorities?: ICertificateAuthorityPartial[];
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  isDirty?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const CustomESTForm = ({
  formData,
  certAuthorities,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  isDirty = true,
  onChange,
  onSubmit,
  onCancel,
}: ICustomESTFormProps) => {
  const validationsConfig = useMemo(() => {
    return generateFormValidations(certAuthorities ?? [], isEditing);
  }, [certAuthorities, isEditing]);

  const validations = useMemo(() => {
    return validateFormData(formData, validationsConfig);
  }, [formData, validationsConfig]);

  const { name, url, username, password } = formData;

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
          helpText="Letters, numbers, and underscores only."
        />
        <InputField
          label="URL"
          name="url"
          value={url}
          error={validations.url?.message}
          onChange={onChange}
          parseTarget
          placeholder="https://example.com/well-known/est/abc123"
        />
        <InputField
          label="Username"
          name="username"
          value={username}
          error={validations.username?.message}
          onChange={onChange}
          parseTarget
          helpText="The username used to authenticate with the EST endpoint."
        />
        <InputField
          type="password"
          label="Password"
          name="password"
          value={password}
          error={validations.password?.message}
          onChange={onChange}
          parseTarget
          helpText="The password used to authenticate with the EST endpoint."
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

export default CustomESTForm;
