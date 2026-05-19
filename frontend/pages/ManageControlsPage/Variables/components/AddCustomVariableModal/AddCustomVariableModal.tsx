import React, { useContext, useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IVariablePayload } from "interfaces/variables";
import { hasStatusKey } from "interfaces/errors";
import variablesAPI from "services/entities/variables";
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
  const [variableName, setVariableName] = useState("");
  const [variableValue, setVariableValue] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const [
    formValidation,
    setFormValidation,
  ] = useState<IAddCustomVariableFormValidation>(() =>
    validateFormData({ name: variableName, value: variableValue })
  );

  const onInputChange = (update: { name: string; value: string }) => {
    const name = update.name;
    let value = update.value;
    if (name === "name") {
      value = value.trimEnd().toUpperCase();
      setVariableName(value);
    } else if (name === "value") {
      setVariableValue(value);
    }
    setFormValidation(
      validateFormData({
        name: variableName,
        value: variableValue,
        [update.name]: value,
      })
    );
  };

  const onClickSave = async (name: string, value: string) => {
    const validation = validateFormData({ name, value }, true);
    if (validation.isValid) {
      setIsSaving(true);
      const newVariable: IVariablePayload = {
        name: variableName,
        value: variableValue,
      };
      try {
        await variablesAPI.addVariable(newVariable);
        renderFlash("success", "Variable created.");
        onSave();
      } catch (error) {
        if (hasStatusKey(error) && error.status === 409) {
          renderFlash("error", "A variable with this name already exists.");
        } else {
          renderFlash(
            "error",
            "An error occurred while saving the variable. Please try again."
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
          value={variableName}
          label="Name"
          name="name"
          parseTarget
          helpText={
            <span>
              You can use this in your script or configuration profile as
              &ldquo;$FLEET_SECRET_{variableName}&rdquo;.
            </span>
          }
          error={formValidation.name?.message}
        />
        <InputField
          onChange={onInputChange}
          value={variableValue}
          label="Value"
          name="value"
          parseTarget
          error={formValidation.value?.message}
        />
        <div className="modal-cta-wrap">
          <Button
            onClick={() => {
              onClickSave(variableName, variableValue);
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
