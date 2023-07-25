import React, { useState } from "react";

import { IResetPasswordFormErrors } from "interfaces/user";

import Button from "components/buttons/Button";
import InputFieldWithIconStories from "../fields/InputFieldWithIcon/InputFieldWithIcon.stories";
import validate from "components/forms/ResetPasswordForm/validate";

const baseClass = "reset-password-form";

export interface IFormData {
  new_password: string;
  new_password_confirmation: string;
}

interface IResetPasswordFormProps {
  serverErrors: string;
  handleSubmit: (formData: IFormData) => void;
  onChangeFunction: () => void;
  editPasswordErrors: IResetPasswordFormErrors;
}
const ResetPasswordForm = ({
  serverErrors,
  handleSubmit,
  onChangeFunction,
  editPasswordErrors,
}: IResetPasswordFormProps): JSX.Element => {
  const [errors, setErrors] = useState<any>(editPasswordErrors);
  const [formData, setFormData] = useState<any>({
    new_password: "",
    new_password_confirmation: "",
  });

  const onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      setErrors({
        ...errors,
        [formField]: null,
      });
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
        error={errors.name}
        label="Confirm password"
        placeholder="Confirm password"
        onChange={onInputChange("new_password_confirmation")}
        value={formData.new_password_confirmation || ""}
        className={`${baseClass}__input`}
        type="password"
      />
      <div className={`${baseClass}__button-wrap`}>
        <Button
          variant="brand"
          onClick={handleSubmit}
          className={`${baseClass}__btn`}
          type="submit"
        >
          Reset password
        </Button>
      </div>
    </form>
  );
};

export default ResetPasswordForm;
