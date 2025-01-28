import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { AppContext } from "context/app";
import { ILabelSummary } from "interfaces/label";

import DataError from "components/DataError";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import AddSoftwareVppForm from "./AddSoftwareVppForm";
import { teamHasVPPToken } from "./helpers";

const baseClass = "software-app-store-vpp";
//
interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareAppStoreVpp = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
  const { isPremiumTier } = useContext(AppContext);

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

    return (
      <div className={`${baseClass}__content`}>
        <AddSoftwareVppForm
          labels={labels || []}
          router={router}
          teamId={currentTeamId}
          hasVppToken={hasVppToken}
          noVppTokenUploaded={noVppTokenUploaded}
          vppApps={vppApps}
        />
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreVpp;
