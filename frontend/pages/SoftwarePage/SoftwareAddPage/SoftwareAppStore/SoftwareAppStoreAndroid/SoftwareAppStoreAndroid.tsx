import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { ILabelSummary } from "interfaces/label";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";

import { getPathWithQueryParams } from "utilities/url";
import SoftwareAndroidForm from "pages/SoftwarePage/components/forms/SoftwareAndroidForm";
import { getErrorMessage } from "./helpers";
import { ISoftwareVppFormData } from "../../../components/forms/SoftwareVppForm/SoftwareVppForm";

const baseClass = "software-app-store-android";

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareAppStoreAndroid = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier } = useContext(AppContext);

  const [isLoading, setIsLoading] = useState(false);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);

  const {
    data: vppInfo,
    isLoading: isLoadingVppInfo,
    error: errorVppInfo,
  } = useQuery<IGetVppTokensResponse, AxiosError>(
    ["vppInfo", currentTeamId],
    () => mdmAppleAPI.getVppTokens(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 30000,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
    }
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
      staleTime: 10000,
    }
  );

  const goBackToSoftwareTitles = (showAvailableForInstallOnly = false) => {
    const queryParams = {
      team_id: currentTeamId,
      ...(showAvailableForInstallOnly && { available_for_install: true }),
    };

    router.push(getPathWithQueryParams(PATHS.SOFTWARE_TITLES, queryParams));
  };

  const onClickPreviewEndUserExperience = () => {
    setShowPreviewEndUserExperience(!showPreviewEndUserExperience);
  };

  const onAddSoftware = async (formData: ISoftwareVppFormData) => {
    if (!formData.selectedApp) {
      return;
    }

    setIsLoading(true);

    try {
      const {
        software_title_id: softwareVppTitleId,
      } = await mdmAppleAPI.addVppApp(currentTeamId, formData);

      renderFlash(
        "success",
        <>
          <b>{formData.selectedApp.name}</b> successfully added.
        </>,
        { persistOnPageChange: true }
      );

      router.push(
        getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(softwareVppTitleId.toString()),
          { team_id: currentTeamId }
        )
      );
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }

    setIsLoading(false);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return (
        <PremiumFeatureMessage className={`${baseClass}__premium-message`} />
      );
    }

    if (isLoadingVppInfo || isLoadingLabels) {
      return <Spinner />;
    }

    return (
      <div className={`${baseClass}__content`}>
        <SoftwareAndroidForm
          onSubmit={onAddSoftware}
          onCancel={goBackToSoftwareTitles}
          onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          isLoading={isLoading}
        />
        {showPreviewEndUserExperience && (
          <CategoriesEndUserExperienceModal
            onCancel={onClickPreviewEndUserExperience}
          />
        )}
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreAndroid;
