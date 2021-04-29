import React, { useState, useCallback } from "react";

import Modal from "components/modals/Modal";
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

const EditTeamModal = (props: IEditTeamModalProps): JSX.Element => {
  const { onCancel, onSubmit, defaultName } = props;

  const [name, setName] = useState(defaultName);

  const onInputChange = useCallback(
    (value: string) => {
      setName(value);
    },
    [setName]
  );

  const onFormSubmit = useCallback(() => {
    onSubmit({
      name,
    });
  }, [onSubmit, name]);

  return (
    <Modal title={"Edit team"} onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <InputFieldWithIcon
          autofocus
          // error={errors.name}
          name="name"
          onChange={onInputChange}
          placeholder="Team name"
          value={name}
        />
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
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
