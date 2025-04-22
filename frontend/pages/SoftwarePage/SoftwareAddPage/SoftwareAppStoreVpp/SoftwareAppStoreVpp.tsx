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
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Button from "components/buttons/Button";

import { getPathWithQueryParams } from "utilities/url";
import SoftwareVppForm from "./SoftwareVppForm";
import { getErrorMessage, teamHasVPPToken } from "./helpers";
import { ISoftwareVppFormData } from "./SoftwareVppForm/SoftwareVppForm";

const baseClass = "software-app-store-vpp";
//

interface IEnableVppMessage {
  onEnableVpp: () => void;
}

const EnableVppMessage = ({ onEnableVpp }: IEnableVppMessage) => (
  <div className={`${baseClass}__enable-vpp-message`}>
    <p className={`${baseClass}__enable-vpp-title`}>
      Volume Purchasing Program (VPP) isn&apos;t enabled
    </p>
    <p className={`${baseClass}__enable-vpp-description`}>
      To add App Store apps, first enable VPP.
    </p>
    <Button onClick={onEnableVpp}>Enable VPP</Button>
  </div>
);

interface IAddTeamToVppMessage {
  onEditVpp: () => void;
}

const AddTeamToVppMessage = ({ onEditVpp }: IAddTeamToVppMessage) => (
  <div className={`${baseClass}__enable-vpp-message`}>
    <p className={`${baseClass}__enable-vpp-title`}>
      This team isn&apos;t added to Volume Purchasing Program (VPP)
    </p>
    <p className={`${baseClass}__enable-vpp-description`}>
      To add App Store apps, first add this team to VPP.
    </p>
    <Button onClick={onEditVpp}>Edit VPP</Button>
  </div>
);

const NoVppAppsMessage = () => (
  <div className={`${baseClass}__no-vpp-message`}>
    <p className={`${baseClass}__no-vpp-title`}>
      You don&apos;t have any App Store apps
    </p>
    <p className={`${baseClass}__no-vpp-description`}>
      You must purchase apps in{" "}
      <CustomLink
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/abm-apps`}
        text="ABM"
        newTab
      />
      .<br />
      App Store apps that are already added to this team are not listed.
    </p>
  </div>
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
  const { isPremiumTier } = useContext(AppContext);

  const [isLoading, setIsLoading] = useState(false);
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

  const goBackToSoftwareTitles = (showAvailableForInstallOnly = false) => {
    const queryParams = {
      team_id: currentTeamId,
      ...(showAvailableForInstallOnly && { available_for_install: true }),
    };

    router.push(getPathWithQueryParams(PATHS.SOFTWARE_TITLES, queryParams));
  };

  const onAddSoftware = async (formData: ISoftwareVppFormData) => {
    if (!formData.selectedApp) {
      return;
    }

    setIsLoading(true);

    try {
      await mdmAppleAPI.addVppApp(currentTeamId, formData);
      renderFlash(
        "success",
        <>
          <b>{formData.selectedApp.name}</b> successfully added.
        </>,
        { persistOnPageChange: true }
      );

      goBackToSoftwareTitles(true);
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
      return <DataError className={`${baseClass}__error`} />;
    }

    if (noVppTokenUploaded) {
      return (
        <EnableVppMessage
          onEnableVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
        />
      );
    }

    if (!hasVppToken) {
      return (
        <AddTeamToVppMessage
          onEditVpp={() => router.push(PATHS.ADMIN_INTEGRATIONS_VPP)}
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
          onCancel={goBackToSoftwareTitles}
          isLoading={isLoading}
          vppApps={vppApps}
        />
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreVpp;
