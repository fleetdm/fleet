import React, { useCallback, useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IFormField } from "interfaces/form_field";

export interface IConfirmInviteFormData {
  name: string;
  password: string;
  password_confirmation: string;
}
interface IConfirmInviteFormProps {
  className: string;
  defaultFormData: Partial<IConfirmInviteFormData>;
  handleSubmit: (data: IConfirmInviteFormData) => void;
}
const ConfirmInviteForm = ({
  className,
  defaultFormData,
  handleSubmit,
}: IConfirmInviteFormProps) => {
  const [formData, setFormData] = useState<IConfirmInviteFormData>({
    name: defaultFormData.name || "",
    password: defaultFormData.password || "",
    password_confirmation: defaultFormData.password || "",
  });

  const { name, password, password_confirmation } = formData;

  const onInputChange = ({ name: n, value }: IFormField) => {
    const newFormData = { ...formData, [n]: value };
    setFormData(newFormData);
  };

  const onSubmit = useCallback(() => {
    handleSubmit(formData);
  }, [formData, handleSubmit]);

  return (
    <form className={className} autoComplete="off">
      <InputField
        label="Full name"
        autofocus
        onChange={onInputChange}
        name="name"
        value={name}
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
        parseTarget
      />
      <InputField
        label="Confirm password"
        type="password"
        placeholder="Confirm password"
        onChange={onInputChange}
        name="password_confirmation"
        value={password_confirmation}
        parseTarget
      />
      <Button
        onClick={onSubmit}
        className="confirm-invite-button"
        variant="brand"
      >
        Submit
      </Button>
    </form>
  );
};

export default ConfirmInviteForm;
