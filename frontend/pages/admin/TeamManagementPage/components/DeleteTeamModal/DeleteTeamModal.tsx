import React from 'react';


// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Modal from 'components/modals/Modal';
import Button from 'components/buttons/Button';

const baseClass = 'delete-team-modal';

interface IDeleteTeamModalProps {
  name: string;
  onSubmit: () => void;
  onCancel: () => void;
}

const DeleteTeamModal = (props: IDeleteTeamModalProps): JSX.Element => {
  const { name, onSubmit, onCancel } = props;

  return (
    <Modal
      title={'Delete team'}
      onExit={onCancel}
      className={baseClass}
    >
      <form className={`${baseClass}__form`}>
        <p>You are about to delete <span className={`${baseClass}__name`}>{name}</span> from Fleet.</p>
        <p>Members of this team who are not members of other teams will lose access to Fleet.</p>
        <p className={`${baseClass}__warning`}>This action cannot be undone.</p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="alert"
            onClick={onSubmit}
          >
            Delete
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

export default DeleteTeamModal;
