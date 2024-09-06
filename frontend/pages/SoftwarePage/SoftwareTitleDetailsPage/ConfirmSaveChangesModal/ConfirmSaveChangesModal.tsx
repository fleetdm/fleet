import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "save-changes-modal";

export interface IConfirmSaveChangesModalProps {
  onSaveChanges: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  softwarePackageName?: string;
  onClose: () => void;
}

const ConfirmSaveChangesModal = ({
  onSaveChanges,
  softwarePackageName,
  onClose,
}: IConfirmSaveChangesModalProps) => {
  const warningText = (
    <>
      The changes you are making will cancel any pending installs and uninstalls
      {softwarePackageName ? (
        <>
          {" "}
          for <b> {softwarePackageName}</b>
        </>
      ) : (
        ""
      )}
      .
    </>
  );
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
            // isLoading={isUpdating} TODO
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
