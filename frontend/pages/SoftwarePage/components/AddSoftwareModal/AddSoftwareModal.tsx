import React, { useState } from "react";

import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import AddSoftwareForm from "../AddSoftwareForm";
import { IAddSoftwareFormData } from "../AddSoftwareForm/AddSoftwareForm";

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
  const [isUploading, setIsUploading] = useState(false);

  const onAddSoftware = (formData: IAddSoftwareFormData) => {
    console.log("formData", formData);
    // setIsUploading(true);
  };

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
          <AddSoftwareForm
            isUploading={isUploading}
            onCancel={onExit}
            onSubmit={onAddSoftware}
          />
        )}
      </>
    </Modal>
  );
};

export default AddSoftwareModal;
