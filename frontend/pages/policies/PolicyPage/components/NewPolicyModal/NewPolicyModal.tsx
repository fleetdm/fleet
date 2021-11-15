import React, { useState } from "react";
import { size } from "lodash";

import { IPolicyFormData } from "interfaces/policy";
import { useDeepEffect } from "utilities/hooks";

import Checkbox from "components/forms/fields/Checkbox"; // @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface INewPolicyModalProps {
  baseClass: string;
  queryValue: string;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  setIsSaveModalOpen: (isOpen: boolean) => void;
}

const validatePolicyName = (name: string) => {
  const errors: { [key: string]: any } = {};

  if (!name) {
    errors.name = "Policy name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const NewPolicyModal = ({
  baseClass,
  queryValue,
  onCreatePolicy,
  setIsSaveModalOpen,
}: INewPolicyModalProps) => {
  const [name, setName] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [errors, setErrors] = useState<{ [key: string]: any }>({});

  useDeepEffect(() => {
    if (name) {
      setErrors({});
    }
  }, [name]);

  const handleUpdate = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    const { valid, errors: newErrors } = validatePolicyName(name);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      onCreatePolicy({
        query_description: description,
        query_name: name,
        query: queryValue,
      });

      setIsSaveModalOpen(false);
    }
  };

  return (
    <Modal title={"Save policy"} onExit={() => setIsSaveModalOpen(false)}>
      <form className={`${baseClass}__save-modal-form`} autoComplete="off">
        <InputField
          name="name"
          onChange={(value: string) => setName(value)}
          value={name}
          error={errors.name}
          inputClassName={`${baseClass}__policy-save-modal-name`}
          label="Name"
          placeholder="What is your policy called?"
        />
        <InputField
          name="description"
          onChange={(value: string) => setDescription(value)}
          value={description}
          inputClassName={`${baseClass}__policy-save-modal-description`}
          label="Description"
          type="textarea"
          placeholder="What information does your query reveal?"
        />
        <hr />
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--modal`}
        >
          <Button
            className={`${baseClass}__btn`}
            onClick={() => setIsSaveModalOpen(false)}
            variant="text-link"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={handleUpdate}
          >
            Save policy
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default NewPolicyModal;
