import React, { useState, useContext, useEffect } from "react";
import { size } from "lodash";

import { PolicyContext } from "context/policy";
import { IPlatformSelector } from "hooks/usePlaformSelector";
import { IPolicyFormData } from "interfaces/policy";
import { IPlatformString } from "interfaces/platform";
import { useDeepEffect } from "utilities/hooks";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import ReactTooltip from "react-tooltip";

export interface INewPolicyModalProps {
  baseClass: string;
  queryValue: string;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  setIsNewPolicyModalOpen: (isOpen: boolean) => void;
  backendValidators: { [key: string]: string };
  platformSelector: IPlatformSelector;
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
  onCreatePolicy,
  setIsNewPolicyModalOpen,
  backendValidators,
  platformSelector,
}: INewPolicyModalProps): JSX.Element => {
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryResolution,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const [name, setName] = useState<string>(lastEditedQueryName);
  const [description, setDescription] = useState<string>(
    lastEditedQueryDescription
  );
  const [resolution, setResolution] = useState<string>(
    lastEditedQueryResolution
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  const disableSave = !platformSelector.isAnyPlatformSelected;

  useDeepEffect(() => {
    if (name) {
      setErrors({});
    }
  }, [name]);

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const handleSavePolicy = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const newPlatformString = platformSelector
      .getSelectedPlatforms()
      .join(",") as IPlatformString;
    setLastEditedQueryPlatform(newPlatformString);

    const { valid: validName, errors: newErrors } = validatePolicyName(name);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (!disableSave && validName) {
      onCreatePolicy({
        description,
        name,
        query: queryValue,
        resolution,
        platform: newPlatformString,
      });
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
        {platformSelector.render()}
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--modal`}
        >
          <Button
            className={`${baseClass}__button--modal-cancel`}
            onClick={() => setIsNewPolicyModalOpen(false)}
            variant="text-link"
          >
            Cancel
          </Button>
          <span
            className={`${baseClass}__button-wrap--modal-save`}
            data-tip
            data-for={`${baseClass}__button--modal-save-tooltip`}
            data-tip-disable={!disableSave}
          >
            <Button
              className={`${baseClass}__button--modal-save`}
              type="submit"
              variant="brand"
              onClick={handleSavePolicy}
              disabled={disableSave}
            >
              Save
            </Button>
            <ReactTooltip
              className={`${baseClass}__button--modal-save-tooltip`}
              place="bottom"
              type="dark"
              effect="solid"
              id={`${baseClass}__button--modal-save-tooltip`}
              backgroundColor="#3e4771"
            >
              Select the platform(s) this
              <br />
              policy will be checked on
              <br />
              to save the policy.
            </ReactTooltip>
          </span>
        </div>
      </form>
    </Modal>
  );
};

export default NewPolicyModal;
