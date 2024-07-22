import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";
import { AxiosResponse } from "axios";

import PATHS from "router/paths";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";
import { QueryParams, buildQueryStringFromParams } from "utilities/url";
import { IApiError } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import AddSoftwareForm from "../AddSoftwareForm";
import { IAddSoftwareFormData } from "../AddSoftwareForm/AddSoftwareForm";
import { getErrorMessage } from "./helpers";

// 8 minutes + 15 seconds to account for extra roundtrip time.
const UPLOAD_TIMEOUT = (8 * 60 + 15) * 1000;
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

    // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
    try {
      await softwareAPI.addSoftwarePackage(formData, teamId, UPLOAD_TIMEOUT);
      renderFlash(
        "success",
        <>
          <b>{formData.software?.name}</b> successfully added.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
      onExit();

      const newQueryParams: QueryParams = { team_id: teamId };
      if (formData.selfService) {
        newQueryParams.self_service = true;
      } else {
        newQueryParams.available_for_install = true;
      }

      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(newQueryParams)}`
      );
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
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
