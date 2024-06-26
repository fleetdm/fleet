import React, { useState, useCallback, useEffect } from "react";

import { ITeamFormData } from "services/entities/teams";

import Modal from "components/Modal";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";

const baseClass = "edit-team-modal";

interface IRenameTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: ITeamFormData) => void;
  defaultName: string;
  backendValidators: { [key: string]: string };
  isUpdatingTeams: boolean;
}

const RenameTeamModal = ({
  onCancel,
  onSubmit,
  defaultName,
  backendValidators,
  isUpdatingTeams,
}: IRenameTeamModalProps): JSX.Element => {
  const [name, setName] = useState(defaultName);
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

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit({ name });
  };

  return (
    <Modal title="Rename team" onExit={onCancel} className={baseClass}>
      <form
        className={`${baseClass}__form`}
        onSubmit={onFormSubmit}
        autoComplete="off"
      >
        <InputField
          autofocus
          name="name"
          onChange={onInputChange}
          label="Team name"
          placeholder="Team name"
          value={name}
          error={errors.name}
          ignore1password
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            disabled={name === ""}
            className="save-loading"
            isLoading={isUpdatingTeams}
          >
            Save
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default RenameTeamModal;
