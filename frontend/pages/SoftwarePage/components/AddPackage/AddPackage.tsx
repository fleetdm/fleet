import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";

import { getErrorReason } from "interfaces/errors";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import softwareAPI from "services/entities/software";
import { QueryParams, buildQueryStringFromParams } from "utilities/url";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";

import AddPackageForm from "../AddPackageForm";
import { IAddPackageFormData } from "../AddPackageForm/AddPackageForm";
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
  setAddedSoftwareToken: (token: string) => void;
}

const AddPackage = ({
  teamId,
  router,
  onExit,
  setAddedSoftwareToken,
}: IAddPackageProps) => {
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

  const onAddPackage = async (formData: IAddPackageFormData) => {
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
      // any unique string - triggers SW refetch
      setAddedSoftwareToken(`${Date.now()}`);
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(newQueryParams)}`
      );
    } catch (e) {
      const reason = getErrorReason(e);
      // TODO - confirm message matching
      if (
        reason.includes("Couldn't add. Fleet couldn't read the version from")
      ) {
        renderFlash(
          "error",
          `${reason}. ${(
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
              text="Learn more"
            />
          )} `
        );
      }
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
