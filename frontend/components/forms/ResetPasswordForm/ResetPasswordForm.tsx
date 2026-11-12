import React from "react";

import Button from "components/buttons/Button";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import {
  validatePassword,
  isEqual,
  isPresent,
} from "components/forms/validators";
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

  if (!isPresent(newPassword)) {
    errors.new_password = "Enter a new password";
  } else {
    const { error_code: errorCode, error } = validatePassword(newPassword);
    if (errorCode) {
      errors.new_password = PASSWORD_ERRORS[errorCode] || error;
    }
  }

  if (!isPresent(newPasswordConfirmation)) {
    errors.new_password_confirmation = "Confirm your new password";
  } else if (newPassword && !isEqual(newPassword, newPasswordConfirmation)) {
    errors.new_password_confirmation = "Match the password above";
  }

  return errors;
};

const ResetPasswordForm = ({
  handleSubmit,
}: IResetPasswordFormProps): JSX.Element => {
  const {
    getTextFieldProps,
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
        {...getTextFieldProps("new_password")}
        autofocus
        label="New password"
        placeholder="New password"
        className={`${baseClass}__input`}
        type="password"
        helpText="12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#)."
        ignore1Password={false}
        disabled={isSubmitting}
      />
      <InputFieldWithIcon
        {...getTextFieldProps("new_password_confirmation")}
        label="Confirm password"
        placeholder="Confirm password"
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
