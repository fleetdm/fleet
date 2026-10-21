import { AxiosError } from "axios";
import React, { useContext, useState } from "react";
import { useQuery, useQueryClient } from "react-query";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { ILabelSummary } from "interfaces/label";
import { formatVppTokenLabel } from "interfaces/mdm";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import PATHS from "router/paths";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import softwareAPI from "services/entities/software";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import SoftwareVppForm from "../../../components/forms/SoftwareVppForm";
import { ISoftwareVppFormData } from "../../../components/forms/SoftwareVppForm/SoftwareVppForm";

import {
  getAssignedVppTokens,
  getErrorMessage,
  getFleetVppTokens,
} from "./helpers";

const baseClass = "software-app-store-vpp";
//

interface IEnableVppMessage {
  onEnableVpp: () => void;
  isGlobalAdmin?: boolean;
}

const EnableVppMessage = ({
  onEnableVpp,
  isGlobalAdmin,
}: IEnableVppMessage) => (
  <EmptyState
    variant="list"
    header="Volume Purchasing Program (VPP) isn't enabled"
    info={
      isGlobalAdmin
        ? "Enable VPP to add App Store apps (MDM required)."
        : "To add App Store apps, ask your admin to enable VPP."
    }
    primaryButton={
      isGlobalAdmin ? (
        <Button onClick={onEnableVpp}>Enable VPP</Button>
      ) : undefined
    }
  />
);

interface IAddTeamToVppMessage {
  onEditVpp: () => void;
  isGlobalAdmin?: boolean;
}

const AddTeamToVppMessage = ({
  onEditVpp,
  isGlobalAdmin,
}: IAddTeamToVppMessage) => (
  <EmptyState
    variant="list"
    header="This fleet isn't added to Volume Purchasing Program (VPP)"
    info={
      isGlobalAdmin
        ? "To add App Store apps, first add this fleet to VPP."
        : "To add App Store apps, ask your admin to add this fleet to VPP."
    }
    primaryButton={
      isGlobalAdmin ? <Button onClick={onEditVpp}>Edit VPP</Button> : undefined
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
          text="Apple Business"
          newTab
        />
        <br />
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
  const {
    isPremiumTier,
    isGlobalAdmin,
    isMacMdmEnabledAndConfigured,
  } = useContext(AppContext);
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
      enabled: !!isMacMdmEnabledAndConfigured,
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
  const assignedTokens = getAssignedVppTokens(
    currentTeamId,
    vppInfo?.vpp_tokens
  );
  const fleetTokens = getFleetVppTokens(currentTeamId, vppInfo?.vpp_tokens);
  const [selectedTokenId, setSelectedTokenId] = useState<number>();
  const vppTokenId =
    fleetTokens.find((t) => t.id === selectedTokenId)?.id ?? fleetTokens[0]?.id;

  const {
    data: vppApps,
    isLoading: isLoadingVppApps,
    error: errorVppApps,
  } = useQuery(
    ["vppSoftware", currentTeamId, vppTokenId],
    () => mdmAppleAPI.getVppApps(currentTeamId, vppTokenId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!vppTokenId && !!isMacMdmEnabledAndConfigured,
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
      } = await softwareAPI.addAppStoreApp(currentTeamId, {
        ...formData,
        vppTokenId,
      });

      notify.success(
        <>
          <b>{formData.selectedApp.name}</b> successfully added.
        </>
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

    // VPP needs Apple MDM; a token uploaded before MDM was turned off must not be used.
    if (!isMacMdmEnabledAndConfigured) {
      return (
        <EnableVppMessage
          onEnableVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
          isGlobalAdmin={isGlobalAdmin}
        />
      );
    }

    if (isLoadingVppInfo || isLoadingLabels) {
      return <Spinner />;
    }

    if (errorVppInfo || isErrorLabels) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    if (noVppTokenUploaded) {
      return (
        <EnableVppMessage
          onEnableVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
          isGlobalAdmin={isGlobalAdmin}
        />
      );
    }

    if (!assignedTokens.length) {
      return (
        <AddTeamToVppMessage
          onEditVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
          isGlobalAdmin={isGlobalAdmin}
        />
      );
    }

    // Every token assigned to this fleet has expired.
    if (!fleetTokens.length) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    // The token dropdown stays above the list so a token that is loading,
    // errored, or empty doesn't strand the user on it.
    const renderApps = () => {
      if (isLoadingVppApps) {
        return <Spinner />;
      }
      if (errorVppApps) {
        return <DataError verticalPaddingSize="pad-xxxlarge" />;
      }
      if (!vppApps) {
        return <NoVppAppsMessage />;
      }
      return (
        <SoftwareVppForm
          key={vppTokenId}
          labels={labels || []}
          onSubmit={onAddSoftware}
          onCancel={goBackToSoftwareLibrary}
          onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          isLoading={isLoading}
          vppApps={vppApps}
        />
      );
    };

    return (
      <div className={`${baseClass}__content`}>
        {fleetTokens.length > 1 && (
          <DropdownWrapper
            ariaLabel="Select VPP token"
            name="vpp-token"
            label="VPP token"
            onChange={(token) =>
              token && setSelectedTokenId(Number(token.value))
            }
            value={vppTokenId?.toString()}
            options={fleetTokens.map((t) => ({
              label: formatVppTokenLabel(t),
              value: t.id.toString(),
              helpText: t.org_name,
            }))}
          />
        )}
        {renderApps()}
        {showPreviewEndUserExperience && (
          <CategoriesEndUserExperienceModal
            onCancel={onClickPreviewEndUserExperience}
            teamId={currentTeamId}
            isIosOrIpadosApp={isIosOrIpadosApp}
          />
        )}
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreVpp;
