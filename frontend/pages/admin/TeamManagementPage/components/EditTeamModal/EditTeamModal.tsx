import React, { useState, useCallback, useEffect } from "react";

import { ITeamFormData } from "services/entities/teams";

import Modal from "components/Modal";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "edit-team-modal";

interface IEditTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: ITeamFormData) => void;
  defaultName: string;
  backendValidators: { [key: string]: string };
  isLoading: boolean;
}

const EditTeamModal = ({
  onCancel,
  onSubmit,
  defaultName,
  backendValidators,
  isLoading,
}: IEditTeamModalProps): JSX.Element => {
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
    <Modal title={"Edit team"} onExit={onCancel} className={baseClass}>
      {isLoading ? (
        <Spinner />
      ) : (
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
          />
          <div className={`${baseClass}__btn-wrap`}>
            <Button
              className={`${baseClass}__btn`}
              type="submit"
              variant="brand"
              disabled={name === ""}
            >
              Save
            </Button>
            <Button
              className={`${baseClass}__btn`}
              onClick={onCancel}
              variant="inverse"
            >
              Cancel
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
};

export default EditTeamModal;
