import React, { useCallback, useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";
import { AxiosProgressEvent } from "axios";

import { FileDetails } from "components/FileUploader/FileUploader";
import { FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME } from "utilities/file/fileUtils";

import { getErrorReason } from "interfaces/errors";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import softwareAPI from "services/entities/software";
import { QueryParams, buildQueryStringFromParams } from "utilities/url";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";

import PackageForm from "../PackageForm";
import { IPackageFormData } from "../PackageForm/PackageForm";
import { getErrorMessage } from "../AddSoftwareModal/helpers";

const baseClass = "add-package";

// 8 minutes + 15 seconds to account for extra roundtrip time.
export const UPLOAD_TIMEOUT = (8 * 60 + 15) * 1000;
export const MAX_FILE_SIZE_MB = 500;
export const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

interface IAddPackageProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
  setAddedSoftwareToken: (token: string) => void;
  setHideTabs: (hideTabs: boolean) => void;
}

const AddPackage = ({
  teamId,
  router,
  onExit,
  setAddedSoftwareToken,
  setHideTabs,
}: IAddPackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);
  const [filename, setFilename] = useState<string | null>(null);

  const [progress, setProgress] = useState(0);

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

  const onAddPackage = useCallback(
    async (formData: IAddSoftwareFormData) => {
      setIsUploading(true);
      setFilename(formData.software?.name || "");

      if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
        renderFlash(
          "error",
          `Couldn't add. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
        );
        onExit();
        setIsUploading(false);
        return;
      }

      setHideTabs(true);

      // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
      try {
        await softwareAPI.addSoftwarePackage({
          data: formData,
          teamId,
          timeout: UPLOAD_TIMEOUT,
          onUploadProgress: (progressEvent: AxiosProgressEvent) => {
            console.log(progressEvent);
            setProgress(progressEvent.progress || 0);
          },
        });
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
          `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(
            newQueryParams
          )}`
        );
      } catch (e) {
        renderFlash("error", getErrorMessage(e));
      }

      onExit();
      setIsUploading(false);
    },
    [onExit, renderFlash, router, setAddedSoftwareToken, teamId]
  );

  const parts = filename?.split(".");
  const ext = parts?.slice(-1)[0] || "";
  console.log(ext);
  const name = parts?.slice(0, -1).join(".") || "";
  const platform = FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME[ext];

  return (
    <div className={baseClass}>
      {!!progress && (
        <FileDetails details={{ name, platform }} progress={progress} />
      )}
      {!progress && (
        <AddPackageForm
          isUploading={isUploading}
          onCancel={onExit}
          onSubmit={onAddPackage}
        />
      )}
    </div>
  );
};

export default AddPackage;
