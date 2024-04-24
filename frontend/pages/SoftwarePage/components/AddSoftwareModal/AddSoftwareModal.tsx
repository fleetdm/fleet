import React, { useState } from "react";

import Modal from "components/Modal";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Button from "components/buttons/Button";
import AddSoftwareForm from "../AddSoftwareForm";

const baseClass = "add-software-modal";

interface IAllTeamsMessageProps {
  onExit: () => void;
}

const AllTeamsMessage = ({ onExit }: IAllTeamsMessageProps) => {
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

interface IAddSoftwareModalProps {
  teamId: number;
  onExit: () => void;
}

const AddSoftwareModal = ({ teamId, onExit }: IAddSoftwareModalProps) => {
  const [isUploading, setIsUploading] = useState(true);

  return (
    <Modal
      title="Add software"
      onExit={onExit}
      width="large"
      className={baseClass}
    >
      <>
        {teamId === APP_CONTEXT_ALL_TEAMS_ID ? (
          <AllTeamsMessage onExit={onExit} />
        ) : (
          <AddSoftwareForm isUploading={isUploading} />
        )}
      </>
    </Modal>
  );
};

export default AddSoftwareModal;
