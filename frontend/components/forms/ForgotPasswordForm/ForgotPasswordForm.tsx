import React from "react";

import Button from "components/buttons/Button";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import validateEmail from "components/forms/validators/valid_email";
import validatePresence from "components/forms/validators/validate_presence";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

const baseClass = "forgot-password-form";

export interface IForgotPasswordFormData {
  email: string;
}

interface IForgotPasswordFormProps {
  handleSubmit: (formData: IForgotPasswordFormData) => void | Promise<unknown>;
}

const validate = ({ email }: IForgotPasswordFormData): IFormErrors => {
  const errors: IFormErrors = {};

  if (!validatePresence(email)) {
    errors.email = "Enter your email";
  } else if (!validateEmail(email)) {
    errors.email = "Enter a valid email";
  }

  return errors;
};

const ForgotPasswordForm = ({
  handleSubmit,
}: IForgotPasswordFormProps): JSX.Element => {
  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IForgotPasswordFormData>({
    initialFormData: { email: "" },
    validate,
  });

  return (
    <form
      onSubmit={onSubmit(handleSubmit)}
      className={baseClass}
      autoComplete="off"
      noValidate
    >
      <p>
        Enter your email below to receive an email with instructions to reset
        your password.
      </p>
      <InputFieldWithIcon
        autofocus
        name="email"
        label="Email"
        placeholder="Email"
        type="email"
        value={formData.email}
        onChange={(value) => setField("email", value)}
        onFocus={() => clearFieldError("email")}
        onBlur={() => validateField("email")}
        error={getError("email")}
        ignore1Password={false}
        disabled={isSubmitting}
      />
      <div className="button-wrap--center">
        <Button
          className={`${baseClass}__submit-btn`}
          type="submit"
          isLoading={isSubmitting}
          disabled={isSubmitting}
          size="wide"
        >
          Get instructions
        </Button>
      </div>
    </form>
  );
};

export default ForgotPasswordForm;
