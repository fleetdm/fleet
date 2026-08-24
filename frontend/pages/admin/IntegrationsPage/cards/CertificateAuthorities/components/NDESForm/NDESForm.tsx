import React from "react";

import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { validateFormData } from "./helpers";

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
  isDirty?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const NDESForm = ({
  formData,
  submitBtnText,
  isSubmitting,
  isDirty = true,
  onChange,
  onSubmit,
  onCancel,
}: INDESFormProps) => {
  // derived from the formData prop (not kept in state) because the parent can
  // change fields this form didn't touch, e.g. the edit modal clears an
  // unchanged password when the SCEP URL, admin URL, or username changes
  const formValidation = validateFormData(formData);

  const { scepURL, adminURL, username, password } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  return (
    <form onSubmit={onSubmitForm}>
      <InputField
        label="SCEP URL"
        name="scepURL"
        value={scepURL}
        error={formValidation.scepURL?.message}
        onChange={onChange}
        parseTarget
        placeholder="https://example.com/certsrv/mscep/mscep.dll"
        helpText="The URL used by client devices to request and retrieve certificates."
      />
      <InputField
        label="Admin URL"
        name="adminURL"
        value={adminURL}
        error={formValidation.adminURL?.message}
        onChange={onChange}
        parseTarget
        placeholder="https://example.com/certsrv/mscep_admin/"
        helpText={
          <>
            The URL for the <b>Network Device Enrollment Service</b> page to
            view configuration details. Okta calls this the <b>Challenge URL</b>
            .
          </>
        }
      />
      <InputField
        label="Username"
        name="username"
        value={username}
        onChange={onChange}
        parseTarget
        placeholder="username@example.microsoft.com"
        helpText="For NDES, this is the username in the down-level logon name
        format required to log in to the SCEP admin page. Okta generates this for you."
      />
      <InputField
        label="Password"
        name="password"
        value={password}
        type="password"
        onChange={onChange}
        parseTarget
        blockAutoComplete
        helpText={
          <>
            For NDES, the password required to log in to the{" "}
            <b>Network Device Enrollment Service</b> page. Okta generates this
            for you.
          </>
        }
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
            type="submit"
            isLoading={isSubmitting}
            disabled={!formValidation.isValid || isSubmitting || !isDirty}
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

export default NDESForm;
