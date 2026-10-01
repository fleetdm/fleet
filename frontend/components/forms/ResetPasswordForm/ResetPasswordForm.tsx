import React from "react";

import Button from "components/buttons/Button";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import validatePassword from "components/forms/validators/valid_password";
import validateEquality from "components/forms/validators/validate_equality";
import validatePresence from "components/forms/validators/validate_presence";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { IResetPasswordForm } from "interfaces/user";

const baseClass = "reset-password-form";

export interface IFormData {
  new_password: string;
  new_password_confirmation: string;
}

interface IResetPasswordFormProps {
  handleSubmit: (formData: IFormData) => void | Promise<unknown>;
}

const PASSWORD_ERRORS: Record<string, string> = {
  too_short: "Enter a password with at least 12 characters",
  too_long: "Enter a password with 48 characters or fewer",
  invalid_format: "Enter a password with at least 1 number and 1 symbol",
};

const validate = (formData: IFormData): IFormErrors => {
  const errors: IFormErrors = {};
  const {
    new_password: newPassword,
    new_password_confirmation: newPasswordConfirmation,
  } = formData;

  if (!validatePresence(newPassword)) {
    errors.new_password = "Enter a new password";
  } else {
    const { error_code: errorCode, error } = validatePassword(newPassword);
    if (errorCode) {
      errors.new_password = PASSWORD_ERRORS[errorCode] || error;
    }
  }

  if (!validatePresence(newPasswordConfirmation)) {
    errors.new_password_confirmation = "Confirm your new password";
  } else if (
    newPassword &&
    !validateEquality(newPassword, newPasswordConfirmation)
  ) {
    errors.new_password_confirmation = "Match the password above";
  }

  return errors;
};

const ResetPasswordForm = ({
  handleSubmit,
}: IResetPasswordFormProps): JSX.Element => {
  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IResetPasswordForm>({
    initialFormData: {
      new_password: "",
      new_password_confirmation: "",
    },
    validate,
    skipTrim: ["new_password", "new_password_confirmation"],
  });

  return (
    <form className={baseClass} onSubmit={onSubmit(handleSubmit)}>
      <InputFieldWithIcon
        error={getError("new_password")}
        autofocus
        name="new_password"
        label="New password"
        placeholder="New password"
        onChange={(value) => setField("new_password", value)}
        onFocus={() => clearFieldError("new_password")}
        onBlur={() => validateField("new_password")}
        value={formData.new_password}
        className={`${baseClass}__input`}
        type="password"
        helpText="12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#)."
        ignore1Password={false}
        disabled={isSubmitting}
      />
      <InputFieldWithIcon
        error={getError("new_password_confirmation")}
        name="new_password_confirmation"
        label="Confirm password"
        placeholder="Confirm password"
        onChange={(value) => setField("new_password_confirmation", value)}
        onFocus={() => clearFieldError("new_password_confirmation")}
        onBlur={() => validateField("new_password_confirmation")}
        value={formData.new_password_confirmation}
        className={`${baseClass}__input`}
        type="password"
        ignore1Password={false}
        disabled={isSubmitting}
      />
      <div className="button-wrap--center">
        <Button
          type="submit"
          size="wide"
          isLoading={isSubmitting}
          disabled={isSubmitting}
        >
          Reset password
        </Button>
      </div>
    </form>
  );
};

export default ResetPasswordForm;
