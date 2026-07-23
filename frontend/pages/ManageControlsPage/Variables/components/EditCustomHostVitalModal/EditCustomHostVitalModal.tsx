import React, { useState } from "react";
import { useMutation } from "react-query";

import { hasStatusKey } from "interfaces/errors";
import { ICustomHostVital } from "interfaces/custom_host_vitals";
import customHostVitalsAPI from "services/entities/custom_host_vitals";
import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";

import {
  validateFormData,
  ICustomHostVitalFormValidation,
  CUSTOM_HOST_VITAL_NAME_MAX_LENGTH,
} from "../../helpers";

const baseClass = "edit-custom-host-vital-modal";

interface IEditCustomHostVitalModalProps {
  vital: ICustomHostVital;
  onCancel: () => void;
  onSave: () => void;
}

const EditCustomHostVitalModal = ({
  vital,
  onCancel,
  onSave,
}: IEditCustomHostVitalModalProps) => {
  const [name, setName] = useState(vital.name);
  const [
    formValidation,
    setFormValidation,
  ] = useState<ICustomHostVitalFormValidation>(() =>
    validateFormData({ name: vital.name })
  );

  const { mutate: updateCustomHostVital, isLoading: isSaving } = useMutation(
    () =>
      customHostVitalsAPI.updateCustomHostVital(vital.id, {
        name: name.trim(),
      }),
    {
      onSuccess: () => {
        notify.success("Custom host vital updated.");
        onSave();
      },
      onError: (error) => {
        if (hasStatusKey(error) && error.status === 409) {
          notify.error("Couldn't save. Host vital name must be unique.", {
            response: error,
          });
        } else {
          notify.error(
            "An error occurred while updating the custom host vital. Please try again.",
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
    updateCustomHostVital();
  };

  return (
    <Modal
      title="Edit custom host vital"
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
          inputOptions={{ maxLength: CUSTOM_HOST_VITAL_NAME_MAX_LENGTH }}
        />
        <div className="modal-cta-wrap">
          <Button
            onClick={onClickSave}
            disabled={!formValidation.isValid || isSaving}
            isLoading={isSaving}
          >
            Save
          </Button>
          <Button variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditCustomHostVitalModal;
