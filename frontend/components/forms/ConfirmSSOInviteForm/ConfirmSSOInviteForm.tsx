import React, { useState } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";

const baseClass = "confirm-invite-page__form";

interface IConfirmSSOInviteFormProps {
  defaultName?: string;
  email?: string;
  handleSubmit: (name: string) => void;
}

const ConfirmSSOInviteForm = ({
  defaultName = "",
  email = "",
  handleSubmit,
}: IConfirmSSOInviteFormProps) => {
  const [name, setName] = useState(defaultName);
  const [nameError, setNameError] = useState<string | null>(null);

  const onNameChange = (value: string) => {
    setName(value);
    if (nameError && value) setNameError(null);
  };

  const onSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!name) {
      setNameError("Full name must be present");
      return;
    }
    handleSubmit(name);
  };

  return (
    <form onSubmit={onSubmit} className={baseClass} autoComplete="off">
      <InputField label="Email" name="email" value={email} disabled />
      <InputField
        label="Full name"
        autofocus
        onChange={onNameChange}
        name="name"
        value={name}
        error={nameError}
        inputOptions={{ maxLength: 80 }}
      />
      <div className="button-wrap--center">
        <Button type="submit" disabled={!!nameError} size="wide">
          Submit
        </Button>
      </div>
    </form>
  );
};

export default ConfirmSSOInviteForm;
