import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IEditScriptConfirmationModalProps {
  onSave: () => void;
  onCancel: () => void;
  scriptName: string;
}

const EditScriptConfirmationModal = ({
  onSave,
  onCancel,
  scriptName,
}: IEditScriptConfirmationModalProps): JSX.Element => {
  return (
    <Modal title="Save changes?" onExit={onCancel} onEnter={onSave}>
      <>
        <p>
          The changes you are making will cancel any pending script runs for{" "}
          <b> {scriptName} </b>
        </p>
        <p>If this script is currently running on a host, it will complete.</p>
        <p>You cannot undo this action.</p>
        <div className="modal-cta-wrap">
          <Button onClick={onSave}>Save</Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default EditScriptConfirmationModal;
