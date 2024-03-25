import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "save-changes-modal";

export interface IConfirmSaveChangesModalProps {
  isUpdating: boolean;
  onSaveChanges: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClose: () => void;
  showChangedSQLCopy?: boolean;
}

const ConfirmSaveChangesModal = ({
  isUpdating,
  onSaveChanges,
  onClose,
  showChangedSQLCopy = false,
}: IConfirmSaveChangesModalProps) => {
  const warningText = showChangedSQLCopy
    ? "Changing this query's SQL will delete its previous results, since the existing report does not reflect the updated query."
    : "The changes you are making to this query will delete its previous results.";

  return (
    <Modal title="Save changes?" onExit={onClose}>
      <form className={`${baseClass}__form`}>
        <p>{warningText}</p>
        <p>You cannot undo this action.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSaveChanges}
            variant="brand"
            className="save-loading"
            isLoading={isUpdating}
          >
            Save
          </Button>
          <Button onClick={onClose} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ConfirmSaveChangesModal;
