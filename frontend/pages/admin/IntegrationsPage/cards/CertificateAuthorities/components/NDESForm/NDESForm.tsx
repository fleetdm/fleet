import React, { useState } from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { INDESFormValidation, validateFormData } from "./helpers";

const baseClass = "ndes-form";

export interface INDESFormData {
  scepURL: string;
  adminURL: string;
  username: string;
  password: string;
}

interface INDESFormProps {
  formData: INDESFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const NDESForm = ({
  formData,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  onChange,
  onSubmit,
  onCancel,
}: INDESFormProps) => {
  const [formValidation, setFormValidation] = useState<INDESFormValidation>(
    () => validateFormData(formData)
  );

  const { scepURL, adminURL, username, password } = formData;

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
    <form onSubmit={onSubmitForm}>
      <div className={`${baseClass}__fields`}>
        <InputField
          label="SCEP URL"
          name="scepURL"
          value={scepURL}
          error={formValidation.scepURL?.message}
          onChange={onInputChange}
          parseTarget
          placeholder="https://example.com/certsrv/mscep/mscep.dll"
          helpText="The URL used by client devices to request and retrieve certificates."
        />
        <InputField
          label="Admin URL"
          name="adminURL"
          value={adminURL}
          error={formValidation.adminURL?.message}
          onChange={onInputChange}
          parseTarget
          placeholder="https://example.com/certsrv/mscep_admin/"
          helpText="The admin interface for managing the SCEP service and viewing configuration details."
        />
        <InputField
          label="Username"
          name="username"
          value={username}
          onChange={onInputChange}
          parseTarget
          placeholder="username@example.microsoft.com"
          helpText="The username in the down-level logon name format required to log in to the SCEP admin page."
        />
        <InputField
          label="Password"
          name="password"
          value={password}
          type="password"
          onChange={onInputChange}
          parseTarget
          blockAutoComplete
          helpText="The password required to log in to the SCEP admin page."
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
            type="submit"
            isLoading={isSubmitting}
            disabled={!formValidation.isValid || isSubmitting}
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

export default NDESForm;
