import React from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import {
  isEqual,
  isPresent,
  validatePassword,
} from "components/forms/validators";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

const baseClass = "change-password-form";

export interface IChangePasswordFormData {
  old_password: string;
  new_password: string;
  new_password_confirmation: string;
}

interface IChangePasswordFormProps {
  handleSubmit: (formData: IChangePasswordFormData) => void | Promise<unknown>;
  onCancel: () => void;
}

const PASSWORD_ERRORS: Record<string, string> = {
  too_short: "Enter a password with at least 12 characters",
  too_long: "Enter a password with 48 characters or fewer",
  invalid_format: "Enter a password with at least 1 number and 1 symbol",
};

const validate = (formData: IChangePasswordFormData): IFormErrors => {
  const errors: IFormErrors = {};
  const {
    old_password: oldPassword,
    new_password: newPassword,
    new_password_confirmation: newPasswordConfirmation,
  } = formData;

  if (!isPresent(oldPassword)) {
    errors.old_password = "Enter your current password";
  }

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

const ChangePasswordForm = ({
  handleSubmit,
  onCancel,
}: IChangePasswordFormProps): JSX.Element => {
  const {
    getTextFieldProps,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IChangePasswordFormData>({
    initialFormData: {
      old_password: "",
      new_password: "",
      new_password_confirmation: "",
    },
    validate,
    skipTrim: ["old_password", "new_password", "new_password_confirmation"],
  });

  return (
    <form onSubmit={onSubmit(handleSubmit)} className={baseClass}>
      <InputField
        {...getTextFieldProps("old_password")}
        autofocus
        label="Current password"
        type="password"
        ignore1password={false}
        disabled={isSubmitting}
      />
      <InputField
        {...getTextFieldProps("new_password")}
        label="New password"
        type="password"
        helpText="12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#)."
        ignore1password={false}
        disabled={isSubmitting}
      />
      <InputField
        {...getTextFieldProps("new_password_confirmation")}
        label="Confirm password"
        type="password"
        ignore1password={false}
        disabled={isSubmitting}
      />
      <div className="modal-cta-wrap">
        <Button type="submit" isLoading={isSubmitting} disabled={isSubmitting}>
          Change password
        </Button>
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default ChangePasswordForm;
