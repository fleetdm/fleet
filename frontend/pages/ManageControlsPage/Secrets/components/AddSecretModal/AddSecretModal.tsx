import React, { useContext, useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ISecretPayload } from "interfaces/secrets";
import secretsAPI from "services/entities/secrets";
import { NotificationContext } from "context/notification";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { useFormValidation } from "hooks/useFormValidation";
import ADD_SECRET_VALIDATIONS from "./helpers";

const baseClass = "fleet-add-secret-modal";

interface AddSecretModalProps {
  onCancel: () => void;
  onSave: () => void;
}

export interface IAddSecretFormData {
  name: string;
  value: string;
}

const AddSecretModal = ({ onCancel, onSave }: AddSecretModalProps) => {
  const [isSaving, setIsSaving] = useState(false);
  const { renderFlash } = useContext(NotificationContext);

  const {
    formData,
    isValid,
    getFieldError,
    setField,
    validateAll,
    handleSubmit,
  } = useFormValidation<IAddSecretFormData>({
    initialFormData: { name: "", value: "" },
    validationConfig: ADD_SECRET_VALIDATIONS,
  });

  const onInputChange = (update: { name: string; value: string }) => {
    const processedValue =
      update.name === "name"
        ? update.value.trimEnd().toUpperCase()
        : update.value;
    setField(update.name as keyof IAddSecretFormData, processedValue);
  };

  const onSaveSecret = async (data: IAddSecretFormData) => {
    setIsSaving(true);
    const newSecret: ISecretPayload = {
      name: data.name,
      value: data.value,
    };
    try {
      await secretsAPI.addSecret(newSecret);
      renderFlash("success", "Variable created.");
      onSave();
    } catch (error: unknown) {
      const apiError = error as { status?: number };
      if (apiError.status === 409) {
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
  };

  return (
    <Modal title="Add custom variable" onExit={onCancel} className={baseClass}>
      <form
        className={`${baseClass}__add-secret-form`}
        onSubmit={handleSubmit(onSaveSecret)}
      >
        <InputField
          onChange={onInputChange}
          onBlur={validateAll}
          value={formData.name}
          label="Name"
          name="name"
          parseTarget
          helpText={
            <span>
              You can use this in your script or configuration profile as
              &ldquo;$FLEET_SECRET_{formData.name}&rdquo;.
            </span>
          }
          error={getFieldError("name")}
        />
        <InputField
          onChange={onInputChange}
          onBlur={validateAll}
          value={formData.value}
          label="Value"
          name="value"
          parseTarget
          error={getFieldError("value")}
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            disabled={!isValid || isSaving}
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
