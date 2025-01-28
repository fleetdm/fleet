import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "save-changes-modal";

export interface IConfirmSaveChangesModalProps {
  onSaveChanges: () => void;
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
        <p>
          Installs or uninstalls currently running on a host will still
          complete, but results won&apos;t appear in Fleet.
        </p>
        <p>You cannot undo this action.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSaveChanges}
            variant="brand"
            className="save-loading"
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
