import React, { FormEvent, useState } from "react";
import { size } from "lodash";

import { IResetPasswordFormErrors } from "interfaces/user";

import Button from "components/buttons/Button";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import validatePresence from "components/forms/validators/validate_presence";
import validatePassword from "components/forms/validators/valid_password";
import validateEquality from "components/forms/validators/validate_equality";

const baseClass = "reset-password-form";

export interface IFormData {
  new_password: string;
  new_password_confirmation: string;
}

interface IResetPasswordFormProps {
  serverErrors: string;
  handleSubmit: (formData: IFormData) => void;
}
const ResetPasswordForm = ({
  serverErrors,
  handleSubmit,
}: IResetPasswordFormProps): JSX.Element => {
  const [errors, setErrors] = useState<IResetPasswordFormErrors>({});
  const [formData, setFormData] = useState<any>({
    new_password: "",
    new_password_confirmation: "",
  });

  const validate = (): boolean => {
    const {
      new_password: newPassword,
      new_password_confirmation: newPasswordConfirmation,
    } = formData;

    const noMatch =
      newPassword &&
      newPasswordConfirmation &&
      !validateEquality(newPassword, newPasswordConfirmation);

    if (!validatePassword(newPassword)) {
      setErrors({
        ...errors,
        new_password: "Password must meet the criteria below",
      });
    }

    if (!validatePresence(newPasswordConfirmation)) {
      setErrors({
        ...errors,
        new_password_confirmation:
          "New password confirmation field must be completed",
      });
    }

    if (!validatePresence(newPassword)) {
      setErrors({
        ...errors,
        new_password: "New password field must be completed",
      });
    }

    if (noMatch) {
      setErrors({
        ...errors,
        new_password_confirmation: "Passwords do not match",
      });
    }

    const valid = !size(errors);

    return valid;
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = validate();

    if (valid) {
      return handleSubmit(formData);
    }
  };

  const onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      setErrors({});
      setFormData({
        ...formData,
        [formField]: value,
      });
    };
  };

  return (
    <form className={baseClass}>
      <InputFieldWithIcon
        error={errors.new_password}
        autofocus
        label="New password"
        placeholder="New password"
        onChange={onInputChange("new_password")}
        value={formData.new_password || ""}
        className={`${baseClass}__input`}
        type="password"
        hint={[
          "Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)",
        ]}
      />
      <InputFieldWithIcon
        error={errors.new_password_confirmation}
        label="Confirm password"
        placeholder="Confirm password"
        onChange={onInputChange("new_password_confirmation")}
        value={formData.new_password_confirmation || ""}
        className={`${baseClass}__input`}
        type="password"
      />
      <div className={`${baseClass}__button-wrap`}>
        <Button
          type="submit"
          variant="brand"
          onClick={onFormSubmit}
          className={`${baseClass}__btn`}
        >
          Reset password
        </Button>
      </div>
    </form>
  );
};

export default ResetPasswordForm;
