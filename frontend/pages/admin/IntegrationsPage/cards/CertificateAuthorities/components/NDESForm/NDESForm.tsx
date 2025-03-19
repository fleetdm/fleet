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
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const NDESForm = ({
  formData,
  submitBtnText,
  isSubmitting,
  onChange,
  onSubmit,
  onCancel,
}: INDESFormProps) => {
  const [formValidation, setFormValidation] = useState<INDESFormValidation>({
    isValid: false,
  });

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
          inputWrapperClass={`${baseClass}__scep-url-input`}
          label="SCEP URL"
          name="scepURL"
          tooltip={
            <>
              The URL used by client devices
              <br /> to request and retrieve certificates.
            </>
          }
          value={scepURL}
          onChange={onInputChange}
          parseTarget
          placeholder="https://example.com/certsrv/mscep/mscep.dll"
        />
        <InputField
          inputWrapperClass={`${baseClass}__admin-url-input`}
          label="Admin URL"
          name="adminURL"
          tooltip={
            <>
              The admin interface for managing the SCEP
              <br /> service and viewing configuration details.
            </>
          }
          value={adminURL}
          onChange={onInputChange}
          parseTarget
          placeholder="https://example.com/certsrv/mscep_admin/"
        />
        <InputField
          inputWrapperClass={`${baseClass}__username-input`}
          label="Username"
          name="username"
          tooltip={
            <>
              The username in the down-level logon name format
              <br />
              required to log in to the SCEP admin page.
            </>
          }
          value={username}
          onChange={onInputChange}
          parseTarget
          placeholder="username@example.microsoft.com"
        />
        <InputField
          inputWrapperClass={`${baseClass}__password-input`}
          label="Password"
          name="password"
          tooltip={
            <>
              The password to use to log in
              <br />
              to the SCEP admin page.
            </>
          }
          value={password}
          type="password"
          onChange={onInputChange}
          parseTarget
          placeholder="••••••••"
          blockAutoComplete
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
