import React, { useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { validateFormData, IAddSecretModalFormValidation } from "./helpers";

const baseClass = "fleet-add-secret-modal";

interface AddSecretModalProps {
  onCancel: () => void;
  onSubmit: (secretName: string, secretValue: string) => Promise<object>;
  isSaving: boolean;
}

export interface IAddSecretModalScheduleFormData {
  name: string;
  value: string;
}

const AddSecretModal = ({
  onCancel,
  onSubmit,
  isSaving,
}: AddSecretModalProps) => {
  const [secretName, setSecretName] = useState("");
  const [secretValue, setSecretValue] = useState("");

  const [
    formValidation,
    setFormValidation,
  ] = useState<IAddSecretModalFormValidation>(() =>
    validateFormData({ name: secretName, value: secretValue })
  );

  const onInputChange = (update: { name: string; value: string }) => {
    const name = update.name;
    let value = update.value;
    if (name === "name") {
      value = value.trimRight().toUpperCase();
      setSecretName(value);
    } else if (name === "value") {
      value = value.trimRight();
      setSecretValue(value);
    }
    setFormValidation(
      validateFormData({
        name: secretName,
        value: secretValue,
        [update.name]: value,
      })
    );
  };

  const onSave = (name: string, value: string) => {
    const validation = validateFormData({ name, value }, true);
    if (validation.isValid) {
      onSubmit(name, value).catch((error) => {
        if (error.status === 409) {
          setFormValidation({
            ...validation,
            name: {
              isValid: false,
              message: "A secret with this name already exists.",
            },
          });
        }
      });
    } else {
      setFormValidation(validation);
    }
  };

  return (
    <Modal title="Add custom variable" onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__add-secret-form`}>
        <InputField
          onChange={onInputChange}
          value={secretName}
          label="Name"
          name="name"
          parseTarget
          helpText={
            <span>
              You can use this in your script or configuration profile as
              &ldquo;$FLEET_SECRET_{secretName}&rdquo;.
            </span>
          }
          error={formValidation.name?.message}
        />
        <InputField
          onChange={onInputChange}
          value={secretValue}
          label="Value"
          name="value"
          parseTarget
          error={formValidation.value?.message}
        />
        <div className="modal-cta-wrap">
          <Button
            onClick={() => {
              onSave(secretName, secretValue);
            }}
            disabled={!formValidation.isValid}
            isLoading={isSaving}
          >
            Save
          </Button>
          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default AddSecretModal;
