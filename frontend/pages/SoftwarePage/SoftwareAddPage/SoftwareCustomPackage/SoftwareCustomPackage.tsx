import React, { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { isCancel } from "axios";

import PATHS from "router/paths";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getFileDetails, IFileDetails } from "utilities/file/fileUtils";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import softwareAPI, {
  MAX_FILE_SIZE_BYTES,
  MAX_FILE_SIZE_MB,
  UPLOAD_TIMEOUT,
} from "services/entities/software";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";

import CustomLink from "components/CustomLink";
import FileProgressModal from "components/FileProgressModal";
import PackageForm from "pages/SoftwarePage/components/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/PackageForm/PackageForm";

import { getErrorMessage } from "./helpers";

const baseClass = "software-custom-package";

interface ISoftwarePackageProps {
  currentTeamId: number;
  router: InjectedRouter;
  isSidePanelOpen: boolean;
  setSidePanelOpen: (isOpen: boolean) => void;
}

const SoftwareCustomPackage = ({
  currentTeamId,
  router,
  isSidePanelOpen,
  setSidePanelOpen,
}: ISoftwarePackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [uploadProgress, setUploadProgress] = React.useState(0);
  const [uploadDetails, setUploadDetails] = React.useState<IFileDetails | null>(
    null
  );

  // TODO: This will cancel the request when the component unmounts. Notably when the back
  // button is clicked, we haven't found a way to showing the native unload prompt. Without this effect,
  // a long running request would continue in the background. What do we want to happen?
  const abortRef = React.useRef(new AbortController());
  useEffect(() => {
    abortRef.current = new AbortController();
    return () => {
      abortRef.current.abort();
    };
  }, []);

  useEffect(() => {
    let timeout: NodeJS.Timeout;

    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Next line with e.returnValue is included for legacy support
      // e.g.Chrome / Edge < 119
      e.returnValue = true;
    };

    // set up event listener to prevent user from leaving page while uploading
    if (uploadDetails) {
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
  }, [uploadDetails]);

  const onCancel = () => {
    router.push(
      `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
        team_id: currentTeamId,
      })}`
    );
  };

  const onSubmit = async (formData: IPackageFormData) => {
    console.log("submit", formData);
    if (!formData.software) {
      renderFlash(
        "error",
        `Couldn't add. Please refresh the page and try again.`
      );
      return;
    }

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't add. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      return;
    }

    setUploadDetails(getFileDetails(formData.software));

    // Note: This TODO is copied to onSaveSoftwareChanges in EditSoftwareModal
    // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
    try {
      abortRef.current = new AbortController();
      await softwareAPI.addSoftwarePackage({
        data: formData,
        teamId: currentTeamId,
        timeout: UPLOAD_TIMEOUT,
        onUploadProgress: (progressEvent) => {
          setUploadProgress(progressEvent.progress || 0);
        },
        signal: abortRef.current.signal,
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

      const newQueryParams: QueryParams = { team_id: currentTeamId };
      if (formData.selfService) {
        newQueryParams.self_service = true;
      } else {
        newQueryParams.available_for_install = true;
      }
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(newQueryParams)}`
      );
    } catch (e) {
      // we don't want to show an error if the request is canceled (e.g. user navigates away)
      if (!isCancel(e)) {
        const reason = getErrorReason(e);
        if (
          reason.includes("Couldn't add. Fleet couldn't read the version from")
        ) {
          renderFlash(
            "error",
            <>
              {reason}{" "}
              <CustomLink
                newTab
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
                text="Learn more"
                iconColor="core-fleet-white"
              />
            </>
          );
        } else {
          renderFlash("error", getErrorMessage(e));
        }
      }
    }
    setUploadDetails(null);
  };

  return (
    <div className={baseClass}>
      <PackageForm
        showSchemaButton={!isSidePanelOpen}
        onClickShowSchema={() => setSidePanelOpen(true)}
        className={`${baseClass}__package-form`}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
      {uploadDetails && (
        <FileProgressModal
          fileDetails={uploadDetails}
          fileProgress={uploadProgress}
        />
      )}
    </div>
  );
};

export default SoftwareCustomPackage;
