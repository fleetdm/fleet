import React, { useContext, useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ISecretPayload } from "interfaces/secrets";
import secretsAPI from "services/entities/secrets";
import { NotificationContext } from "context/notification";
import InputField from "components/forms/fields/InputField";
import { validateFormData, IAddCustomVariableFormValidation } from "./helpers";

const baseClass = "add-custom-variable-modal";

interface AddCustomVariableModalProps {
  onCancel: () => void;
  onSave: () => void;
}

export interface IAddCustomVariableFormData {
  name: string;
  value: string;
}

const AddCustomVariableModal = ({
  onCancel,
  onSave,
}: AddCustomVariableModalProps) => {
  const [secretName, setSecretName] = useState("");
  const [secretValue, setSecretValue] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const [
    formValidation,
    setFormValidation,
  ] = useState<IAddCustomVariableFormValidation>(() =>
    validateFormData({ name: secretName, value: secretValue })
  );

  const onInputChange = (update: { name: string; value: string }) => {
    const name = update.name;
    let value = update.value;
    if (name === "name") {
      value = value.trimRight().toUpperCase();
      setSecretName(value);
    } else if (name === "value") {
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

  const onClickSave = async (name: string, value: string) => {
    const validation = validateFormData({ name, value }, true);
    if (validation.isValid) {
      setIsSaving(true);
      const newSecret: ISecretPayload = {
        name: secretName,
        value: secretValue,
      };
      try {
        await secretsAPI.addSecret(newSecret);
        renderFlash("success", "Variable created.");
        onSave();
      } catch (error: any) {
        if (error.status === 409) {
          renderFlash("error", "A secret with this name already exists.");
        } else {
          renderFlash(
            "error",
            "An error occurred while saving the secret. Please try again."
          );
        }
      } finally {
        setIsSaving(false);
      }
    } else {
      setFormValidation(validation);
    }
  };

  return (
    <Modal title="Add custom variable" onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__add-variable-form`}>
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
              onClickSave(secretName, secretValue);
            }}
            disabled={!formValidation.isValid || isSaving}
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

export default AddCustomVariableModal;
