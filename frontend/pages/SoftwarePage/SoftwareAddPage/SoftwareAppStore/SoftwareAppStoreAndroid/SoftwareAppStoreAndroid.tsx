import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { ILabelSummary } from "interfaces/label";
import SoftwareAndroidForm from "pages/SoftwarePage/components/forms/SoftwareAndroidForm";
import { ISoftwareAndroidFormData } from "pages/SoftwarePage/components/forms/SoftwareAndroidForm/SoftwareAndroidForm";
import PATHS from "router/paths";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import softwareAPI from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

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
  const {
    isPremiumTier,
    isAndroidMdmEnabledAndConfigured,
    isGlobalAdmin,
  } = useContext(AppContext);

  const [isLoading, setIsLoading] = useState(false);

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", currentTeamId],
    () =>
      labelsAPI
        .summary(currentTeamId)
        .then((res) => getCustomLabels(res.labels)),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!isPremiumTier && !!isAndroidMdmEnabledAndConfigured,
      staleTime: 10000,
    }
  );

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

      notify.success(
        <>
          <strong>{softwareTitleName || "Android app"}</strong> successfully
          added.
        </>
      );

      router.push(
        getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(softwareAppStoreTitleId.toString()),
          { fleet_id: currentTeamId }
        )
      );
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
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
          labels={labels || []}
          isLoadingLabels={isLoadingLabels}
          isErrorLabels={isErrorLabels}
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
