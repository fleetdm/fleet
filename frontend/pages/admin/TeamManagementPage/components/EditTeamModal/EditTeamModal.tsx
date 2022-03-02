import React, { useState, useCallback, useEffect } from "react";

import Modal from "components/Modal";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import Button from "components/buttons/Button";

const baseClass = "edit-team-modal";

export interface IEditTeamFormData {
  name: string;
}

interface IEditTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: IEditTeamFormData) => void;
  defaultName: string;
  backendValidators: { [key: string]: string };
}

const EditTeamModal = ({
  onCancel,
  onSubmit,
  defaultName,
  backendValidators,
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
      <form
        className={`${baseClass}__form`}
        onSubmit={onFormSubmit}
        autoComplete="off"
      >
        <InputFieldWithIcon
          autofocus
          name="name"
          onChange={onInputChange}
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
    </Modal>
  );
};

export default EditTeamModal;
