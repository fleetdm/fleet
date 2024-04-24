import React from "react";

import Modal from "components/Modal";
import { API_ALL_TEAMS_ID, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Button from "components/buttons/Button";

const baseClass = "add-software-modal";

interface IAddSoftwareModalProps {
  teamId: number;
  onExit: () => void;
}

const AddSoftwareModal = ({ teamId, onExit }: IAddSoftwareModalProps) => {
  console.log(teamId);
  const renderAllTeamsMessage = () => {
    return (
      <>
        <p>
          Please select a team first. Software can&apos;t be added when{" "}
          <b>All teams</b> is selected.
        </p>
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onExit}>
            Done
          </Button>
        </div>
      </>
    );
  };

  const renderForm = () => {
    return <p>Form</p>;
  };

  return (
    <Modal
      title="Add software"
      onExit={onExit}
      width="large"
      className={baseClass}
    >
      <>
        {teamId === APP_CONTEXT_ALL_TEAMS_ID
          ? renderAllTeamsMessage()
          : renderForm()}
      </>
    </Modal>
  );
};

export default AddSoftwareModal;
