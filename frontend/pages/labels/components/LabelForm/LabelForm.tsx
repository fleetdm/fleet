import React, { ReactNode, useState } from "react";

import validate_presence from "components/forms/validators/validate_presence";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";

export interface ILabelFormData {
  name: string;
  description: string;
}

interface ILabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  additionalFields?: ReactNode;
  isUpdatingLabel?: boolean;
  onCancel: () => void;
  onSave: (formData: ILabelFormData, isValid: boolean) => void;
}

const baseClass = "label-form";

const LabelForm = ({
  defaultName = "",
  defaultDescription = "",
  additionalFields,
  isUpdatingLabel,
  onCancel,
  onSave,
}: ILabelFormProps) => {
  const [name, setName] = useState(defaultName);
  const [description, setDescription] = useState(defaultDescription);
  const [nameError, setNameError] = useState<string | null>("");

  const onNameChange = (value: string) => {
    setName(value);
    setNameError(null);
  };

  const onDescriptionChange = (value: string) => {
    setDescription(value);
  };

  const onSubmitForm = (evt: React.FormEvent) => {
    evt.preventDefault();

    let isFormValid = true;
    if (!validate_presence(name)) {
      setNameError("Label name must be present");
      isFormValid = false;
    }

    onSave({ name, description }, isFormValid);
  };

  return (
    <form className={`${baseClass}__wrapper`} onSubmit={onSubmitForm}>
      <InputField
        error={nameError}
        name="name"
        onChange={onNameChange}
        value={name}
        inputClassName={`${baseClass}__label-title`}
        label="Name"
        placeholder="Label name"
      />
      <InputField
        name="description"
        onChange={onDescriptionChange}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
      />
      {additionalFields}
      <div className="button-wrap">
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
        <Button type="submit" variant="brand" isLoading={isUpdatingLabel}>
          Save
        </Button>
      </div>
    </form>
  );
};

export default LabelForm;
