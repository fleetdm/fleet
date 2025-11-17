import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import axios from "axios";
import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import softwareAPI from "services/entities/software";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import { ISoftwareAndroidFormData } from "pages/SoftwarePage/components/forms/SoftwareAndroidForm/SoftwareAndroidForm";

import { getPathWithQueryParams } from "utilities/url";
import SoftwareAndroidForm from "pages/SoftwarePage/components/forms/SoftwareAndroidForm";
import { getErrorMessage } from "./helpers";
import { ADD_SOFTWARE_ERROR_PREFIX } from "../../helpers";

const baseClass = "software-app-store-android";

const AMAPI_BASE_URL = "https://androidmanagement.googleapis.com/v1/";
const ENTERPRISE_ID = "<your_enterprise_id>"; // Set this appropriately

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

  const onAddSoftware = async (formData: ISoftwareAndroidFormData) => {
    if (!formData.applicationID) {
      return;
    }

    setIsLoading(true);

    try {
      // HANDLED SERVER SIDE?
      // // Validate app ID exists using AMAPI
      // const appId = formData.applicationID; // e.g. "us.zoom.videomeetings"
      // const appApiUrl = `${AMAPI_BASE_URL}enterprises/${ENTERPRISE_ID}/applications/${appId}`;
      // // Requires OAuth access token with the required scopes
      // const accessToken = "<your_oauth_access_token>";

      // const appApiRes = await axios.get(appApiUrl, {
      //   headers: {
      //     Authorization: `Bearer ${accessToken}`,
      //   },
      // });

      // // If successful, app metadata is present
      // if (!appApiRes.data || !appApiRes.data.name) {
      //   renderFlash(
      //     "error",
      //     `${ADD_SOFTWARE_ERROR_PREFIX} The application ID isn't available in Play Store. Please find ID on the Play Store and try again.`
      //   );
      //   setIsLoading(false);
      //   return;
      // }

      const {
        software_title_id: softwareAppStoreTitleId,
        // Maybe this will return name and can render success message with the name?
        name: softwareTitleName,
      } = await softwareAPI.addAppStoreApp(currentTeamId, formData);

      renderFlash(
        "success",
        <>
          {/* <strong>{appApiRes.data.name}</strong> successfully added. */}
          <strong>{softwareTitleName}</strong> successfully added.
        </>,
        { persistOnPageChange: true }
      );

      router.push(
        getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(softwareAppStoreTitleId.toString()),
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
