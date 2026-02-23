import React, { useState, useCallback, useEffect } from "react";

import { ITeamFormData } from "services/entities/teams";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "create-team-modal";

interface ICreateTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: ITeamFormData) => void;
  backendValidators: { [key: string]: string };
  isUpdatingTeams: boolean;
}

const CreateTeamModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  isUpdatingTeams,
}: ICreateTeamModalProps): JSX.Element => {
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
    (evt: any) => {
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
            disabled={name === ""}
            className="create-loading"
            isLoading={isUpdatingTeams}
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

export default CreateTeamModal;
