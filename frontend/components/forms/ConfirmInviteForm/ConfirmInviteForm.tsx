import React from "react";

import validateEquality from "components/forms/validators/validate_equality";
import validatePresence from "components/forms/validators/validate_presence";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";

const baseClass = "confirm-invite-page__form";

export interface IConfirmInviteFormData {
  name: string;
  password: string;
  password_confirmation: string;
}

interface IConfirmInviteFormProps {
  defaultFormData?: Partial<IConfirmInviteFormData>;
  handleSubmit: (data: IConfirmInviteFormData) => void | Promise<unknown>;
  ancestorError?: string;
}

const validate = (formData: IConfirmInviteFormData): IFormErrors => {
  const errors: IFormErrors = {};
  const { name, password, password_confirmation: confirmation } = formData;

  if (!validatePresence(name)) {
    errors.name = "Enter your full name";
  }

  if (!validatePresence(password)) {
    errors.password = "Enter a password";
  }

  if (!validatePresence(confirmation)) {
    errors.password_confirmation = "Confirm your password";
  } else if (password && !validateEquality(password, confirmation)) {
    errors.password_confirmation = "Match the password above";
  }

  return errors;
};

const ConfirmInviteForm = ({
  defaultFormData,
  handleSubmit,
  ancestorError,
}: IConfirmInviteFormProps) => {
  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IConfirmInviteFormData>({
    initialFormData: {
      name: defaultFormData?.name || "",
      password: "",
      password_confirmation: "",
    },
    validate,
    skipTrim: ["password", "password_confirmation"],
  });

  return (
    <form
      onSubmit={onSubmit(handleSubmit)}
      className={baseClass}
      autoComplete="off"
    >
      {ancestorError && <div className="form__base-error">{ancestorError}</div>}
      <InputField
        label="Full name"
        autofocus
        name="name"
        value={formData.name}
        onChange={(value: string) => setField("name", value)}
        onFocus={() => clearFieldError("name")}
        onBlur={() => validateField("name")}
        error={getError("name")}
        inputOptions={{ maxLength: 80 }}
        ignore1password={false}
        disabled={isSubmitting}
      />
      <InputField
        label="Password"
        type="password"
        placeholder="Password"
        helpText="Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)"
        name="password"
        value={formData.password}
        onChange={(value: string) => setField("password", value)}
        onFocus={() => clearFieldError("password")}
        onBlur={() => validateField("password")}
        error={getError("password")}
        ignore1password={false}
        disabled={isSubmitting}
      />
      <InputField
        label="Confirm password"
        type="password"
        placeholder="Confirm password"
        name="password_confirmation"
        value={formData.password_confirmation}
        onChange={(value: string) => setField("password_confirmation", value)}
        onFocus={() => clearFieldError("password_confirmation")}
        onBlur={() => validateField("password_confirmation")}
        error={getError("password_confirmation")}
        ignore1password={false}
        disabled={isSubmitting}
      />
      <div className="button-wrap--center">
        <Button
          type="submit"
          size="wide"
          isLoading={isSubmitting}
          disabled={isSubmitting}
        >
          Submit
        </Button>
      </div>
    </form>
  );
};

export default ConfirmInviteForm;
