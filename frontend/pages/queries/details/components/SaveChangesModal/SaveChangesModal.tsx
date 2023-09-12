import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { ICreateQueryRequestBody } from "interfaces/schedulable_query";

const baseClass = "save-changes-modal";

export interface ISaveChangesModalProps {
  isUpdating: boolean;
  onSaveChanges: (formData: ICreateQueryRequestBody) => void;
  toggleSaveChangesModal: () => void;
  sqlUpdated?: boolean;
}

const SaveChangesModal = ({
  isUpdating,
  onSaveChanges,
  toggleSaveChangesModal,
  sqlUpdated = false,
}: ISaveChangesModalProps): JSX.Element => {
  const warningText = () => {
    if (sqlUpdated) {
      return "Changing this query's SQL will delete its previous results, since the existing report does not reflect the updated query.";
    }
    return "The changes you are making to this query will delete its previous results.";
  };

  return (
    <Modal title={"Save changes?"} onExit={toggleSaveChangesModal}>
      <form className={`${baseClass}__form`}>
        <p>{warningText()}</p>
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
          <Button onClick={toggleSaveChangesModal} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default SaveChangesModal;
