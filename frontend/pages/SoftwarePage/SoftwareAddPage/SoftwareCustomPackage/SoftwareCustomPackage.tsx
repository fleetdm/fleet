import React, { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { isAxiosError } from "axios";

import PATHS from "router/paths";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { getFileDetails, IFileDetails } from "utilities/file/fileUtils";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import softwareAPI, {
  MAX_FILE_SIZE_BYTES,
  MAX_FILE_SIZE_MB,
} from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";

import CustomLink from "components/CustomLink";
import FileProgressModal from "components/FileProgressModal";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

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
  const { isPremiumTier } = useContext(AppContext);
  const [uploadProgress, setUploadProgress] = React.useState(0);
  const [uploadDetails, setUploadDetails] = React.useState<IFileDetails | null>(
    null
  );

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary().then((res) => getCustomLabels(res.labels)),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
    }
  );

  useEffect(() => {
    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Next line with e.returnValue is included for legacy support
      // e.g.Chrome / Edge < 119
      e.returnValue = true;
    };

    // set up event listener to prevent user from leaving page while uploading
    if (uploadDetails) {
      addEventListener("beforeunload", beforeUnloadHandler);
    } else {
      removeEventListener("beforeunload", beforeUnloadHandler);
    }

    // clean up event listener and timeout on component unmount
    return () => {
      removeEventListener("beforeunload", beforeUnloadHandler);
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
      await softwareAPI.addSoftwarePackage({
        data: formData,
        teamId: currentTeamId,
        onUploadProgress: (progressEvent) => {
          const progress = progressEvent.progress || 0;
          // for large uploads it seems to take a bit for the server to finalize its response so we'll keep the
          // progress bar at 97% until the server response is received
          setUploadProgress(Math.max(progress - 0.03, 0.01));
        },
      });

      const newQueryParams: QueryParams = { team_id: currentTeamId };
      if (formData.selfService) {
        newQueryParams.self_service = true;
      } else {
        newQueryParams.available_for_install = true;
      }
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(newQueryParams)}`
      );

      renderFlash(
        "success",
        <>
          <b>{formData.software?.name}</b> successfully added.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
    } catch (e) {
      const isTimeout =
        isAxiosError(e) &&
        (e.response?.status === 504 || e.response?.status === 408);
      const reason = getErrorReason(e);

      if (isTimeout) {
        renderFlash(
          "error",
          `Couldn’t upload. Request timeout. Please make sure your server and load balancer timeout is long enough.`
        );
      } else if (reason.includes("Fleet couldn't read the version from")) {
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
    setUploadDetails(null);
  };

  const renderContent = () => {
    if (isLoadingLabels) {
      return <Spinner />;
    }

    if (isErrorLabels) {
      return <DataError className={`${baseClass}__data-error`} />;
    }

    return (
      <>
        <PackageForm
          labels={labels || []}
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
      </>
    );
  };

  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage className={`${baseClass}__premium-message`} />
    );
  }

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareCustomPackage;
