import React, { useState } from "react";
import { useMutation } from "react-query";

import { hasStatusKey } from "interfaces/errors";
// TODO(#48559): replace mock with live API — swap for
// "services/entities/custom_host_vitals" once CRUD endpoints exist.
import customHostVitalsAPI from "services/entities/custom_host_vitals_mock";
import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";

import { validateFormData, ICustomHostVitalFormValidation } from "./helpers";

const baseClass = "add-custom-host-vital-modal";

interface IAddCustomHostVitalModalProps {
  onCancel: () => void;
  onSave: () => void;
}

const AddCustomHostVitalModal = ({
  onCancel,
  onSave,
}: IAddCustomHostVitalModalProps) => {
  const [name, setName] = useState("");
  const [
    formValidation,
    setFormValidation,
  ] = useState<ICustomHostVitalFormValidation>(() =>
    validateFormData({ name: "" })
  );

  const { mutate: addCustomHostVital, isLoading: isSaving } = useMutation(
    () => customHostVitalsAPI.addCustomHostVital({ name: name.trim() }),
    {
      onSuccess: () => {
        notify.success("Custom host vital created.");
        onSave();
      },
      onError: (error) => {
        if (hasStatusKey(error) && error.status === 409) {
          notify.error("Couldn't save. Host vital name must be unique.", {
            response: error,
          });
        } else {
          notify.error(
            "An error occurred while saving the custom host vital. Please try again.",
            { response: error }
          );
        }
      },
    }
  );

  const onInputChange = (value: string) => {
    setName(value);
    setFormValidation(validateFormData({ name: value }));
  };

  const onClickSave = () => {
    const validation = validateFormData({ name }, true);
    if (!validation.isValid) {
      setFormValidation(validation);
      return;
    }
    addCustomHostVital();
  };

  return (
    <Modal
      title="Add custom host vital"
      onExit={onCancel}
      className={baseClass}
    >
      <form className={`${baseClass}__form`}>
        <InputField
          onChange={onInputChange}
          value={name}
          label="Name"
          name="name"
          error={formValidation.name?.message}
          helpText="This will be the vital's label on the host detail page."
        />
        <div className="modal-cta-wrap">
          <Button
            onClick={onClickSave}
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

export default AddCustomHostVitalModal;
