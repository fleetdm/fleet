import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery, useQueryClient } from "react-query";

import PATHS from "router/paths";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { getFileDetails, IFileDetails } from "utilities/file/fileUtils";
import { getPathWithQueryParams, QueryParams } from "utilities/url";
import softwareAPI from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import { AppContext } from "context/app";
import useBlockNavigation from "hooks/useBlockNavigation";
import useGitOpsMode from "hooks/useGitOpsMode";
import { ILabelSummary } from "interfaces/label";

import { notify } from "components/ToastNotification";
import CustomLink from "components/CustomLink";
import FileProgressModal from "components/FileProgressModal";
import InfoBanner from "components/InfoBanner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";

import PackageForm from "pages/SoftwarePage/components/forms/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";

import { getErrorMessage } from "./helpers";

const baseClass = "software-custom-package";

/** Shared GitOps-mode banner for the custom-package flows. Rendered by this
 * page (single-package add) and by `PackageForm`'s multi-package Add modal. */
export const GitOpsCustomPackageBanner = () => (
  <InfoBanner
    icon="info-outline"
    iconColor="ui-fleet-black-50"
    borderRadius="medium"
  >
    Add custom packages in GitOps mode so Fleet can host your software. After
    adding, copy its SHA-256 hash into your YAML so the next GitOps workflow
    doesn&apos;t delete it.{" "}
    <CustomLink
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/software-yaml`}
      text="YAML docs"
      newTab
    />
  </InfoBanner>
);

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
  const { isPremiumTier } = useContext(AppContext);
  const queryClient = useQueryClient();
  const { gitOpsModeEnabled } = useGitOpsMode("software");

  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadDetails, setUploadDetails] = useState<IFileDetails | null>(null);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);
  const [
    isIpadOrIphoneSoftwareSource,
    setIsIpadOrIphoneSoftwareSource,
  ] = useState(false);

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () =>
      labelsAPI
        .summary(currentTeamId)
        .then((res) => getCustomLabels(res.labels)),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
    }
  );

  // Block tab close / hard navigation while an upload is in flight.
  useBlockNavigation(!!uploadDetails);

  const onClickPreviewEndUserExperience = (isIosOrIpadosApp = false) => {
    setShowPreviewEndUserExperience(!showPreviewEndUserExperience);
    setIsIpadOrIphoneSoftwareSource(isIosOrIpadosApp);
  };

  const onCancel = () => {
    router.push(
      getPathWithQueryParams(PATHS.SOFTWARE_LIBRARY, {
        fleet_id: currentTeamId,
      })
    );
  };

  const onSubmit = async (formData: IPackageFormData) => {
    if (!formData.software) {
      notify.error(`Couldn't add. Please refresh the page and try again.`);
      return;
    }

    setUploadDetails(getFileDetails(formData.software));

    // Note: This TODO is copied to onSaveSoftwareChanges in EditSoftwareModal
    // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
    try {
      const {
        software_package: { title_id: softwarePackageTitleId },
      } = await softwareAPI.addSoftwarePackage({
        data: formData,
        teamId: currentTeamId,
        onUploadProgress: (progressEvent) => {
          const progress = progressEvent.progress || 0;
          // for large uploads it seems to take a bit for the server to finalize its response so we'll keep the
          // progress bar at 97% until the server response is received
          setUploadProgress(Math.max(progress - 0.03, 0.01));
        },
      });

      if (!gitOpsModeEnabled) {
        notify.success(
          <>
            <b>{formData.software?.name}</b> successfully added.
            {formData.selfService
              ? " The end user can install from Fleet Desktop."
              : ""}
          </>
        );
      }

      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-titles" }],
      });
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-library" }],
      });

      const newQueryParams: QueryParams = {
        fleet_id: currentTeamId,
      };
      router.push(
        getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(softwarePackageTitleId.toString()),
          newQueryParams
        )
      );
    } catch (e) {
      notify.error(getErrorMessage(e, formData.software?.name), {
        response: e,
      });
    }
    setUploadDetails(null);
  };

  const renderContent = () => {
    if (isLoadingLabels) {
      return <Spinner />;
    }

    if (isErrorLabels) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    return (
      <>
        {gitOpsModeEnabled && <GitOpsCustomPackageBanner />}
        <PackageForm
          labels={labels || []}
          showSchemaButton={!isSidePanelOpen}
          onClickShowSchema={() => setSidePanelOpen(true)}
          className={`${baseClass}__package-form`}
          onCancel={onCancel}
          onSubmit={onSubmit}
          onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
        />
        {uploadDetails && (
          <FileProgressModal
            fileDetails={uploadDetails}
            fileProgress={uploadProgress}
          />
        )}
        {showPreviewEndUserExperience && (
          <CategoriesEndUserExperienceModal
            onCancel={onClickPreviewEndUserExperience}
            teamId={currentTeamId}
            isIosOrIpadosApp={isIpadOrIphoneSoftwareSource}
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
