import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IEditTeamModal {
  onCancel: () => void;
  onRequest: () => void;
}

const baseClass = "edit-team-modal";

const EditTeamModal = ({
  onCancel,
  onRequest,
}: IEditTeamModal): JSX.Element => {
  return (
    <Modal title="Edit team" onExit={onCancel} className={baseClass}>
      <>
        Cool beans
        <div className="modal-cta-wrap">
          <Button onClick={onRequest} variant="brand">
            Save
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default EditTeamModal;
