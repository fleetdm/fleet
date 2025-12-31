import React, { ReactNode, useState } from "react";

import validate_presence from "components/forms/validators/validate_presence";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TeamNameField from "../TeamNameField/TeamNameField";

export interface ILabelFormData {
  name: string;
  description: string;
}

interface ILabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  additionalFields?: ReactNode;
  isUpdatingLabel?: boolean;
  teamName: string | null;
  onCancel: () => void;
  immutableFields: string[];
  onSave: (formData: ILabelFormData, isValid: boolean) => void;
}

const baseClass = "label-form";

const generateDescriptionHelpText = (immutableFields: string[]) => {
  if (immutableFields.length === 0) {
    return "";
  }

  const SUFFIX =
    "are immutable. To make changes, delete this label and create a new one.";

  if (immutableFields.length === 1) {
    return `Label ${immutableFields[0]} ${SUFFIX}`;
  }

  if (immutableFields.length === 2) {
    // No comma for two items: "queries and platforms"
    return `Label ${immutableFields[0]} and ${immutableFields[1]} ${SUFFIX}`;
  }

  // 3+ items: Oxford comma before "and"
  const allButLast = immutableFields.slice(0, -1).join(", ");
  const last = immutableFields.slice(-1);
  return `Label ${allButLast}, and ${last} ${SUFFIX}`;
};

const LabelForm = ({
  defaultName = "",
  defaultDescription = "",
  additionalFields,
  isUpdatingLabel,
  teamName,
  onCancel,
  onSave,
  immutableFields,
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
      {immutableFields.length > 0 ? (
        <span className={`${baseClass}__help-text`}>
          {generateDescriptionHelpText(immutableFields)}
        </span>
      ) : null}
      {teamName ? <TeamNameField name={teamName} /> : null}
      {additionalFields}
      <div className="button-wrap">
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
        <Button type="submit" isLoading={isUpdatingLabel}>
          Save
        </Button>
      </div>
    </form>
  );
};

export default LabelForm;
