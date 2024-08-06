import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import softwareAPI from "services/entities/software";
import { QueryParams, buildQueryStringFromParams } from "utilities/url";

import AddPackageForm from "../AddPackageForm";
import { IAddSoftwareFormData } from "../AddPackageForm/AddSoftwareForm";
import { getErrorMessage } from "../AddSoftwareModal/helpers";

const baseClass = "add-package";

// 8 minutes + 15 seconds to account for extra roundtrip time.
const UPLOAD_TIMEOUT = (8 * 60 + 15) * 1000;
const MAX_FILE_SIZE_MB = 500;
const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

interface IAddPackageProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
}

const AddPackage = ({ teamId, router, onExit }: IAddPackageProps) => {
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

  const onAddPackage = async (formData: IAddSoftwareFormData) => {
    setIsUploading(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't add. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
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
    }

    onExit();
    setIsUploading(false);
  };

  return (
    <div className={baseClass}>
      <AddPackageForm
        isUploading={isUploading}
        onCancel={onExit}
        onSubmit={onAddPackage}
      />
    </div>
  );
};

export default AddPackage;
