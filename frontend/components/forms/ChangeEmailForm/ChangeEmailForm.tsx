import React from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import { isPresent } from "components/forms/validators";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { IUser } from "interfaces/user";

const baseClass = "change-email-form";

interface IChangeEmailFormState {
  password: string;
}

export type IChangeEmailFormData = Partial<IUser> & { password: string };

interface IChangeEmailFormProps {
  defaultData?: Partial<IUser>;
  handleSubmit: (formData: IChangeEmailFormData) => void | Promise<unknown>;
  onCancel: (evt?: React.MouseEvent<HTMLButtonElement>) => void;
  serverErrors?: IFormErrors | null;
}

const validate = (data: IChangeEmailFormState): IFormErrors => {
  const errors: IFormErrors = {};
  if (!isPresent(data.password)) {
    errors.password = "Enter your password";
  }
  return errors;
};

const ChangeEmailForm = ({
  defaultData,
  handleSubmit,
  onCancel,
  serverErrors,
}: IChangeEmailFormProps): JSX.Element => {
  const {
    getTextFieldProps,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IChangeEmailFormState>({
    initialFormData: { password: "" },
    validate,
    serverErrors,
    skipTrim: ["password"],
  });

  const onValidSubmit = ({ password }: IChangeEmailFormState) =>
    handleSubmit({ ...(defaultData ?? {}), password });

  return (
    <form className={baseClass} onSubmit={onSubmit(onValidSubmit)}>
      To update your email you must confirm your password.
      <InputField
        {...getTextFieldProps("password")}
        autofocus
        label="Password"
        type="password"
        disabled={isSubmitting}
        ignore1password={false}
      />
      <div className="modal-cta-wrap">
        <Button type="submit" isLoading={isSubmitting} disabled={isSubmitting}>
          Submit
        </Button>
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default ChangeEmailForm;
