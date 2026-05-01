import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery, useQueryClient } from "react-query";
import { AxiosError } from "axios";
import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { ILabelSummary } from "interfaces/label";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import softwareAPI from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import EmptyState from "components/EmptyState";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Button from "components/buttons/Button";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";

import { getPathWithQueryParams } from "utilities/url";
import SoftwareVppForm from "../../../components/forms/SoftwareVppForm";
import { getErrorMessage, teamHasVPPToken } from "./helpers";
import { ISoftwareVppFormData } from "../../../components/forms/SoftwareVppForm/SoftwareVppForm";

const baseClass = "software-app-store-vpp";
//

interface IEnableVppMessage {
  onEnableVpp: () => void;
  isAdmin: boolean;
}

const EnableVppMessage = ({ onEnableVpp, isAdmin }: IEnableVppMessage) => (
  <EmptyState
    variant="list"
    header="Volume Purchasing Program (VPP) isn't enabled"
    info={
      isAdmin
        ? "To add App Store apps, first enable VPP."
        : "To add App Store apps, ask your admin to enable VPP."
    }
    primaryButton={
      isAdmin ? <Button onClick={onEnableVpp}>Enable VPP</Button> : undefined
    }
  />
);

interface IAddTeamToVppMessage {
  onEditVpp: () => void;
  isAdmin: boolean;
}

const AddTeamToVppMessage = ({ onEditVpp, isAdmin }: IAddTeamToVppMessage) => (
  <EmptyState
    variant="list"
    header="This fleet isn't added to Volume Purchasing Program (VPP)"
    info={
      isAdmin
        ? "To add App Store apps, first add this fleet to VPP."
        : "To add App Store apps, ask your admin to add this fleet to VPP."
    }
    primaryButton={
      isAdmin ? <Button onClick={onEditVpp}>Edit VPP</Button> : undefined
    }
  />
);

const NoVppAppsMessage = () => (
  <EmptyState
    variant="list"
    header="You don't have any App Store apps"
    info={
      <>
        You must purchase apps in{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/abm-apps`}
          text="ABM"
          newTab
        />
        .<br />
        App Store apps that are already added to this fleet are not listed.
      </>
    }
  />
);

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareAppStoreVpp = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier, isGlobalAdmin, isAnyTeamAdmin } = useContext(
    AppContext
  );
  const isAdmin = !!(isGlobalAdmin || isAnyTeamAdmin);
  const queryClient = useQueryClient();

  const [isLoading, setIsLoading] = useState(false);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);
  const [isIosOrIpadosApp, setIsIosOrIpadosApp] = useState(false);

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
    () =>
      labelsAPI
        .summary(currentTeamId)
        .then((res) => getCustomLabels(res.labels)),

    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      staleTime: 10000,
    }
  );

  const noVppTokenUploaded = !vppInfo || !vppInfo.vpp_tokens.length;
  const hasVppToken = teamHasVPPToken(currentTeamId, vppInfo?.vpp_tokens);

  const {
    data: vppApps,
    isLoading: isLoadingVppApps,
    error: errorVppApps,
  } = useQuery(
    ["vppSoftware", currentTeamId],
    () => mdmAppleAPI.getVppApps(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: hasVppToken,
      staleTime: 30000,
      select: (res) => res.app_store_apps,
    }
  );

  const goBackToSoftwareLibrary = () => {
    router.push(
      getPathWithQueryParams(PATHS.SOFTWARE_LIBRARY, {
        fleet_id: currentTeamId,
      })
    );
  };

  const onClickPreviewEndUserExperience = (iosOrIpadosApp?: boolean) => {
    setShowPreviewEndUserExperience(!showPreviewEndUserExperience);
    setIsIosOrIpadosApp(iosOrIpadosApp || false);
  };

  const onAddSoftware = async (formData: ISoftwareVppFormData) => {
    if (!formData.selectedApp) {
      return;
    }

    setIsLoading(true);

    try {
      const {
        software_title_id: softwareVppTitleId,
      } = await softwareAPI.addAppStoreApp(currentTeamId, formData);

      renderFlash(
        "success",
        <>
          <b>{formData.selectedApp.name}</b> successfully added.
        </>,
        { persistOnPageChange: true }
      );

      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-titles" }],
      });
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-library" }],
      });
      queryClient.invalidateQueries({
        queryKey: ["vppSoftware", currentTeamId],
      });

      router.push(
        getPathWithQueryParams(
          PATHS.SOFTWARE_TITLE_DETAILS(softwareVppTitleId.toString()),
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

    if (isLoadingVppInfo || isLoadingVppApps || isLoadingLabels) {
      return <Spinner />;
    }

    if (errorVppInfo || errorVppApps || isErrorLabels) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    if (noVppTokenUploaded) {
      return (
        <EnableVppMessage
          onEnableVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
          isAdmin={isAdmin}
        />
      );
    }

    if (!hasVppToken) {
      return (
        <AddTeamToVppMessage
          onEditVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
          isAdmin={isAdmin}
        />
      );
    }

    if (!vppApps) {
      return <NoVppAppsMessage />;
    }
    return (
      <div className={`${baseClass}__content`}>
        <SoftwareVppForm
          labels={labels || []}
          onSubmit={onAddSoftware}
          onCancel={goBackToSoftwareLibrary}
          onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          isLoading={isLoading}
          vppApps={vppApps}
        />
        {showPreviewEndUserExperience && (
          <CategoriesEndUserExperienceModal
            onCancel={onClickPreviewEndUserExperience}
            isIosOrIpadosApp={isIosOrIpadosApp}
          />
        )}
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreVpp;
