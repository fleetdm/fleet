import React, { useState, useCallback } from "react";

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
}

const EditTeamModal = ({
  onCancel,
  onSubmit,
  defaultName,
}: IEditTeamModalProps): JSX.Element => {
  const [name, setName] = useState(defaultName);

  const onInputChange = useCallback(
    (value: string) => {
      setName(value);
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
