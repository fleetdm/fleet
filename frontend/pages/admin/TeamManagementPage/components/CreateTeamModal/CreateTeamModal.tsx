import React, { useState, useCallback } from 'react';

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Modal from 'components/modals/Modal';
// @ts-ignore
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
import Button from '../../../../../components/buttons/Button';

const baseClass = 'create-team-modal';

export interface ICreateTeamFormData {
  name: string;
}

interface ICreateTeamModalProps {
  onExit: () => void;
  onSubmit: (formData: ICreateTeamFormData) => void;
}

const CreateTeamModal = (props: ICreateTeamModalProps): JSX.Element => {
  const { onExit, onSubmit } = props;

  const [name, setName] = useState('');

  const onInputChange = useCallback((value: string) => {
    setName(value);
  }, [setName]);

  const onFormSubmit = useCallback(() => {
    onSubmit({
      name,
    });
  }, [onSubmit, name]);

  return (
    <Modal
      title={'Create team'}
      onExit={onExit}
      className={`${baseClass}__create-user-modal`}
    >
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
            Create
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onExit}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateTeamModal;
