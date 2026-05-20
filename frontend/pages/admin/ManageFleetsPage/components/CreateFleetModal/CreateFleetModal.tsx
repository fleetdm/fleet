import React, { useState, useCallback, useEffect } from "react";

import { ITeamFormData as IFleetFormData } from "services/entities/teams";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import InputField from "components/forms/fields/InputField";

const baseClass = "create-fleet-modal";

interface ICreateFleetModalProps {
  onCancel: () => void;
  onSubmit: (formData: IFleetFormData) => void;
  backendValidators: { [key: string]: string };
  isUpdatingFleets: boolean;
}

const CreateFleetModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  isUpdatingFleets,
}: ICreateFleetModalProps): JSX.Element => {
  const [name, setName] = useState("");
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const onInputChange = useCallback(
    (value: string) => {
      setName(value);
      setErrors({});
    },
    [setName]
  );

  const onFormSubmit = useCallback(
    (evt: React.FormEvent<HTMLFormElement>) => {
      evt.preventDefault();
      onSubmit({
        name: name.trim(),
      });
    },
    [onSubmit, name]
  );

  return (
    <Modal title="Create fleet" onExit={onCancel} className={baseClass}>
      <form
        className={`${baseClass}__form`}
        onSubmit={onFormSubmit}
        autoComplete="off"
      >
        <InputField
          autofocus
          name="name"
          onChange={onInputChange}
          onBlur={() => {
            setName(name.trim());
          }}
          label="Fleet name"
          placeholder="Workstations"
          value={name}
          error={errors.name}
          ignore1password
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            disabled={name.trim() === ""}
            className="create-loading"
            isLoading={isUpdatingFleets}
          >
            Create
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateFleetModal;
