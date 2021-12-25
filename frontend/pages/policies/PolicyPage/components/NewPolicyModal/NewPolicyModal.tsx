import React, { useState, useContext } from "react";
import { size } from "lodash";

import { IPolicyFormData } from "interfaces/policy";
import { IQueryPlatform } from "interfaces/query";
import { useDeepEffect } from "utilities/hooks";
import { PolicyContext } from "context/policy";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface INewPolicyModalProps {
  baseClass: string;
  queryValue: string;
  platform: IQueryPlatform;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  setIsNewPolicyModalOpen: (isOpen: boolean) => void;
}

const validatePolicyName = (name: string) => {
  const errors: { [key: string]: string } = {};

  if (!name) {
    errors.name = "Policy name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const NewPolicyModal = ({
  baseClass,
  queryValue,
  platform,
  onCreatePolicy,
  setIsNewPolicyModalOpen,
}: INewPolicyModalProps): JSX.Element => {
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryResolution,
  } = useContext(PolicyContext);

  const [name, setName] = useState<string>(lastEditedQueryName);
  const [description, setDescription] = useState<string>(
    lastEditedQueryDescription
  );
  const [resolution, setResolution] = useState<string>(
    lastEditedQueryResolution
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useDeepEffect(() => {
    if (name) {
      setErrors({});
    }
  }, [name]);

  const handleSavePolicy = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const { valid, errors: newErrors } = validatePolicyName(name);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      onCreatePolicy({
        description,
        name,
        query: queryValue,
        resolution,
        platform,
      });

      setIsNewPolicyModalOpen(false);
    }
  };

  return (
    <Modal title={"Save policy"} onExit={() => setIsNewPolicyModalOpen(false)}>
      <form
        onSubmit={handleSavePolicy}
        className={`${baseClass}__save-modal-form`}
        autoComplete="off"
      >
        <InputField
          name="name"
          onChange={(value: string) => setName(value)}
          value={name}
          error={errors.name}
          inputClassName={`${baseClass}__policy-save-modal-name`}
          label="Name"
          placeholder="What yes or no question does your policy ask about your devices?"
        />
        <InputField
          name="description"
          onChange={(value: string) => setDescription(value)}
          value={description}
          inputClassName={`${baseClass}__policy-save-modal-description`}
          label="Description"
          placeholder="Add a description here"
        />
        <InputField
          name="resolution"
          onChange={(value: string) => setResolution(value)}
          value={resolution}
          inputClassName={`${baseClass}__policy-save-modal-resolution`}
          label="Resolution"
          type="textarea"
          placeholder="What are the steps a device owner should take to resolve a host that fails this policy?"
        />
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--modal`}
        >
          <Button
            className={`${baseClass}__btn`}
            onClick={() => setIsNewPolicyModalOpen(false)}
            variant="text-link"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="submit"
            variant="brand"
            onClick={handleSavePolicy}
          >
            Save
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default NewPolicyModal;
