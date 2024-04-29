import React, { useContext, useEffect, useState } from "react";

import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";
import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import AddSoftwareForm from "../AddSoftwareForm";
import { IAddSoftwareFormData } from "../AddSoftwareForm/AddSoftwareForm";

// 2 minutes
const UPLOAD_TIMEOUT = 120000;

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
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  useEffect(() => {
    let timeout: NodeJS.Timeout;

    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Included for legacy support, e.g. Chrome/Edge < 119
      e.returnValue = true;
    };

    // set up event listener to prevent user from leaving page while uploading
    if (isUploading) {
      addEventListener("beforeunload", beforeUnloadHandler);
      timeout = setTimeout(() => {
        removeEventListener("beforeunload", beforeUnloadHandler);
      }, UPLOAD_TIMEOUT);
    } else {
      removeEventListener("beforeunload", beforeUnloadHandler);
    }

    // clean up event listener and timeout on component unmount
    return () => {
      removeEventListener("beforeunload", beforeUnloadHandler);
      clearTimeout(timeout);
    };
  }, [isUploading]);

  const onAddSoftware = async (formData: IAddSoftwareFormData) => {
    console.log("formData", formData);
    setIsUploading(true);

    try {
      await softwareAPI.addSoftwarePackage(formData, teamId);
      renderFlash("success", "Software added successfully!"); // TODO: change message
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }

    setIsUploading(false);
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
