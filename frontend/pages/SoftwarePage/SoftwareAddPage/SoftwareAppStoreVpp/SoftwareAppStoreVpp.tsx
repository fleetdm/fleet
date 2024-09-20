import React from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import DataError from "components/DataError";
import Spinner from "components/Spinner";

import AddSoftwareVppForm from "./AddSoftwareVppForm";
import { teamHasVPPToken } from "./helpers";

const baseClass = "software-app-store-vpp";

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareAppStoreVpp = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
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
    if (isLoadingVppInfo || isLoadingVppApps) {
      return <Spinner />;
    }

    if (errorVppInfo || errorVppApps) {
      return <DataError className={`${baseClass}__error`} />;
    }

    return (
      <div className={`${baseClass}__content`}>
        <AddSoftwareVppForm
          router={router}
          teamId={currentTeamId}
          hasVppToken={hasVppToken}
          vppApps={vppApps}
        />
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreVpp;
