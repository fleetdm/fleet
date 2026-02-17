import React, { ReactNode, useState } from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TeamNameField from "../TeamNameField/TeamNameField";
import { validateLabelFormData, ILabelFormValidation } from "./helpers";

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
  // this holds only the errors we're currently showing
  const [formValidation, setFormValidation] = useState<ILabelFormValidation>({
    isValid: true,
  });

  const currentData = { name, description };

  const onFormChange = (update: { name: string; value: string }) => {
    const { name: fieldName, value } = update;

    const nextData =
      fieldName === "name"
        ? { name: value, description }
        : { name, description: value };

    if (fieldName === "name") {
      setName(value);
    } else if (fieldName === "description") {
      setDescription(value);
    }

    // full validation for new data
    const fullValidation = validateLabelFormData(nextData);

    setFormValidation((prev) => {
      const next: ILabelFormValidation = { ...prev, isValid: true };

      // start from previous errors
      if (prev.name) next.name = prev.name;
      if (prev.description) next.description = prev.description;

      // ONLY CLEAR existing error on this field if it is now valid.
      // Do NOT set a new error if there wasn't one before.
      if (fieldName === "name") {
        if (prev.name && fullValidation.name?.isValid) {
          next.name = undefined; // clear existing name error
        }
      } else if (fieldName === "description") {
        if (prev.description && fullValidation.description?.isValid) {
          next.description = undefined; // clear existing description error
        }
      }

      // recompute isValid from remaining errors
      const fields = [next.name, next.description];
      next.isValid = fields.every((f) => !f || f.isValid);

      return next;
    });
  };

  const onInputBlur = () => {
    // on blur, show all current errors (set all)
    const fullValidation = validateLabelFormData(currentData);
    setFormValidation(fullValidation);
  };

  const onSubmitForm = (evt: React.FormEvent) => {
    evt.preventDefault();

    // on submit, also show all errors
    const fullValidation = validateLabelFormData(currentData);
    setFormValidation(fullValidation);

    onSave(currentData, fullValidation.isValid);
  };

  return (
    <form className={`${baseClass}__wrapper`} onSubmit={onSubmitForm}>
      <InputField
        error={formValidation.name?.message}
        parseTarget
        name="name"
        onChange={onFormChange}
        onBlur={onInputBlur}
        value={name}
        inputClassName={`${baseClass}__label-title`}
        label="Name"
        placeholder="Label name"
        maxLength={255}
      />
      <InputField
        error={formValidation.description?.message}
        parseTarget
        name="description"
        onChange={onFormChange}
        onBlur={onInputBlur}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
        maxLength={255}
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
        <Button
          type="submit"
          isLoading={isUpdatingLabel}
          disabled={!formValidation.isValid}
        >
          Save
        </Button>
      </div>
    </form>
  );
};

export default LabelForm;
