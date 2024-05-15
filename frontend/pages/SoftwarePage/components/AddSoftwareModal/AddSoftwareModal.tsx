import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";
import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";
import { buildQueryStringFromParams } from "utilities/url";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import AddSoftwareForm from "../AddSoftwareForm";
import { IAddSoftwareFormData } from "../AddSoftwareForm/AddSoftwareForm";

// 2 minutes
const UPLOAD_TIMEOUT = 120000;
const MAX_FILE_SIZE_MB = 500;
const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

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
  router: InjectedRouter;
  onExit: () => void;
}

const AddSoftwareModal = ({
  teamId,
  router,
  onExit,
}: IAddSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  useEffect(() => {
    let timeout: NodeJS.Timeout;

    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Next line with e.returnValue is included for legacy support
      // e.g.Chrome / Edge < 119
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
    setIsUploading(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn’t add. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      onExit();
      setIsUploading(false);
      return;
    }

    try {
      await softwareAPI.addSoftwarePackage(formData, teamId);
      renderFlash(
        "success",
        <>
          <b>{formData.software?.name}</b> successfully added. Go to Host
          details page to install software.
        </>
      );
      onExit();
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
          available_for_install: true,
          team_id: teamId,
        })}`
      );
    } catch (e) {
      renderFlash("error", getErrorReason(e));
      onExit();
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
