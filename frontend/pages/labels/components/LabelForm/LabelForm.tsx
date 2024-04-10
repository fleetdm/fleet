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
      setNameError("Label title must be present");
      isFormValid = false;
    }

    onSave({ name, description }, isFormValid);
  };

  // const renderLabelComponent = (): JSX.Element | null => {
  //   if (!showOpenSchemaActionText) {
  //     return null;
  //   }

  //   return (
  //     <Button variant="text-icon" onClick={onOpenSchemaSidebar}>
  //       <>
  //         <Icon name="info" size="small" />
  //         Show schema
  //       </>
  //     </Button>
  //   );
  // };

  // const isBuiltin =
  //   selectedLabel &&
  //   (selectedLabel.label_type === "builtin" || selectedLabel.type === "status");
  // const aceHelpText = isEdit
  //   ? "Label queries are immutable. To change the query, delete this label and create a new one."
  //   : "";

  // if (isBuiltin) {
  //   return (
  //     <div className={`${baseClass}__wrapper`}>
  //       <h1>Built in labels cannot be edited</h1>
  //     </div>
  //   );
  // }

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
