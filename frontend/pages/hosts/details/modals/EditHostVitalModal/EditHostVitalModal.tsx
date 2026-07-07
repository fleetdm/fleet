import React, { useState } from "react";
import { useMutation } from "react-query";

import { IHostCustomVital } from "interfaces/custom_host_vitals";
import { getErrorReason } from "interfaces/errors";
import customHostVitalsAPI from "services/entities/custom_host_vitals";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import { notify } from "components/ToastNotification";

const baseClass = "edit-host-vital-modal";

interface IEditHostVitalModalProps {
  hostId: number;
  vital: IHostCustomVital;
  onCancel: () => void;
  onSave: () => void;
}

const EditHostVitalModal = ({
  hostId,
  vital,
  onCancel,
  onSave,
}: IEditHostVitalModalProps) => {
  const [value, setValue] = useState(vital.value);

  const { mutate: saveValue, isLoading: isSaving } = useMutation(
    () =>
      customHostVitalsAPI.updateHostCustomHostVitalValue(
        hostId,
        vital.custom_host_vital_id,
        value
      ),
    {
      onSuccess: () => {
        notify.success("Successfully updated custom host vital.");
        onSave();
      },
      onError: (error) => {
        notify.error(
          getErrorReason(error) ||
            "Couldn't update custom host vital. Please try again.",
          { response: error }
        );
      },
    }
  );

  const onSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    saveValue();
  };

  return (
    <Modal title="Edit host vital" onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`} onSubmit={onSubmit}>
        <InputField
          value={value}
          label={vital.name}
          name="value"
          onChange={setValue}
        />
        <div className="modal-cta-wrap">
          <Button type="submit" isLoading={isSaving}>
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

export default EditHostVitalModal;
