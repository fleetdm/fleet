import React, { useCallback, useState } from "react";

import validateEquality from "components/forms/validators/validate_equality";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IFormField } from "interfaces/form_field";

const baseClass = "confirm-invite-page__form";
export interface IConfirmInviteFormData {
  name: string;
  password: string;
  password_confirmation: string;
}
interface IConfirmInviteFormProps {
  defaultFormData?: Partial<IConfirmInviteFormData>;
  handleSubmit: (data: IConfirmInviteFormData) => void;
  ancestorError?: string;
}
interface IConfirmInviteFormErrors {
  name?: string | null;
  password?: string | null;
  password_confirmation?: string | null;
}

const validate = (formData: IConfirmInviteFormData) => {
  const errors: IConfirmInviteFormErrors = {};
  const {
    name,
    password,
    password_confirmation: passwordConfirmation,
  } = formData;

  if (!name) {
    errors.name = "Full name must be present";
  }

  if (
    password &&
    passwordConfirmation &&
    !validateEquality(password, passwordConfirmation)
  ) {
    errors.password_confirmation =
      "Password confirmation does not match password";
  }

  if (!password) {
    errors.password = "Password must be present";
  }

  if (!passwordConfirmation) {
    errors.password_confirmation = "Password confirmation must be present";
  }

  return errors;
};
const ConfirmInviteForm = ({
  defaultFormData,
  handleSubmit,
  ancestorError,
}: IConfirmInviteFormProps) => {
  const [formData, setFormData] = useState<IConfirmInviteFormData>({
    name: defaultFormData?.name || "",
    password: defaultFormData?.password || "",
    password_confirmation: defaultFormData?.password || "",
  });
  const [formErrors, setFormErrors] = useState<IConfirmInviteFormErrors>({});

  const { name, password, password_confirmation } = formData;

  const onInputChange = ({ name: n, value }: IFormField) => {
    const newFormData = { ...formData, [n]: value };
    setFormData(newFormData);
    const newErrs = validate(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set on submit
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      // @ts-ignore
      if (newErrs[k]) {
        // @ts-ignore
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onSubmit = useCallback(() => {
    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    handleSubmit(formData);
  }, [formData, handleSubmit]);

  return (
    <form className={baseClass} autoComplete="off">
      {ancestorError && <div className="form__base-error">{ancestorError}</div>}
      <InputField
        label="Full name"
        autofocus
        onChange={onInputChange}
        name="name"
        value={name}
        error={formErrors.name}
        parseTarget
        maxLength={80}
      />
      <InputField
        label="Password"
        type="password"
        placeholder="Password"
        helpText="Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)"
        onChange={onInputChange}
        name="password"
        value={password}
        error={formErrors.password}
        parseTarget
      />
      <InputField
        label="Confirm password"
        type="password"
        placeholder="Confirm password"
        onChange={onInputChange}
        name="password_confirmation"
        value={password_confirmation}
        error={formErrors.password_confirmation}
        parseTarget
      />
      <Button
        onClick={onSubmit}
        disabled={Object.keys(formErrors).length > 0}
        className="confirm-invite-button"
        variant="brand"
      >
        Submit
      </Button>
    </form>
  );
};

export default ConfirmInviteForm;
