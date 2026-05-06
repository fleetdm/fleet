import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import softwareAPI from "services/entities/software";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import EmptyState from "components/EmptyState";
import Button from "components/buttons/Button";
import { ISoftwareAndroidFormData } from "pages/SoftwarePage/components/forms/SoftwareAndroidForm/SoftwareAndroidForm";

import { getPathWithQueryParams } from "utilities/url";
import SoftwareAndroidForm from "pages/SoftwarePage/components/forms/SoftwareAndroidForm";
import { getErrorMessage } from "./helpers";

const baseClass = "software-app-store-android";

interface IEnableAndroidMdmMessageProps {
  onEnableMdm: () => void;
  isGlobalAdmin?: boolean;
}

const EnableAndroidMdmMessage = ({
  onEnableMdm,
  isGlobalAdmin,
}: IEnableAndroidMdmMessageProps) => (
  <EmptyState
    variant="form"
    header="Android MDM isn't enabled"
    info={
      isGlobalAdmin
        ? "To add Android apps, first enable Android MDM."
        : "To add Android apps, ask your admin to enable Android MDM."
    }
    primaryButton={
      isGlobalAdmin ? (
        <Button onClick={onEnableMdm}>Enable Android MDM</Button>
      ) : undefined
    }
  />
);

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareAppStoreAndroid = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    isPremiumTier,
    isAndroidMdmEnabledAndConfigured,
    isGlobalAdmin,
  } = useContext(AppContext);

  const [isLoading, setIsLoading] = useState(false);

  const goBackToSoftwareLibrary = () => {
    router.push(
      getPathWithQueryParams(PATHS.SOFTWARE_LIBRARY, {
        fleet_id: currentTeamId,
      })
    );
  };

  const onEnableAndroidMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_ANDROID);
  };

  const onAddSoftware = async (formData: ISoftwareAndroidFormData) => {
    if (!formData.applicationID) {
      return;
    }

    setIsLoading(true);

    try {
      const {
        software_title_id: softwareAppStoreTitleId,
        name: softwareTitleName,
      } = await softwareAPI.addAppStoreApp(currentTeamId, formData);

      renderFlash(
        "success",
        <>
          <strong>{softwareTitleName || "Android app"}</strong> successfully
          added.
        </>,
        { persistOnPageChange: true }
      );

      router.push(
        getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(softwareAppStoreTitleId.toString()),
          { fleet_id: currentTeamId }
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

    if (!isAndroidMdmEnabledAndConfigured) {
      return (
        <EnableAndroidMdmMessage
          onEnableMdm={onEnableAndroidMdm}
          isGlobalAdmin={isGlobalAdmin}
        />
      );
    }

    return (
      <div className={`${baseClass}__content`}>
        <SoftwareAndroidForm
          onSubmit={onAddSoftware}
          onCancel={goBackToSoftwareLibrary}
          isLoading={isLoading}
        />
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreAndroid;
