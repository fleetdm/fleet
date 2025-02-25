/** software/versions/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";

import softwareAPI, {
  ISoftwareVersionResponse,
  IGetSoftwareVersionQueryKey,
} from "services/entities/software";
import hostsCountAPI, {
  IHostsCountQueryKey,
  IHostsCountResponse,
} from "services/entities/host_count";
import {
  ISoftwareVersion,
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import { ignoreAxiosError } from "interfaces/errors";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/SoftwareVulnerabilitiesTable";
import DetailsNoHosts from "../components/DetailsNoHosts";
import { VulnsNotSupported } from "../components/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";

const baseClass = "software-version-details-page";

interface ISoftwareVersionDetailsRouteParams {
  id: string;
  team_id?: string;
}

type ISoftwareTitleDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareVersionDetailsRouteParams
>;

const SoftwareVersionDetailsPage = ({
  routeParams,
  router,
  location,
}: ISoftwareTitleDetailsPageProps) => {
  const { isPremiumTier, isOnGlobalTeam } = useContext(AppContext);
  const handlePageError = useErrorHandler();

  const versionId = parseInt(routeParams.id, 10);

  const {
    currentTeamId,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
  });

  const {
    data: softwareVersion,
    isLoading: isSoftwareVersionLoading,
    isError: isSoftwareVersionError,
  } = useQuery<
    ISoftwareVersionResponse,
    AxiosError,
    ISoftwareVersion,
    IGetSoftwareVersionQueryKey[]
  >(
    [{ scope: "softwareVersion", versionId, teamId: teamIdForApi }],
    ({ queryKey }) => softwareAPI.getSoftwareVersion(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      select: (data) => data.software,
      onError: (error) => {
        if (!ignoreAxiosError(error, [403, 404])) {
          handlePageError(error);
        }
      },
    }
  );

  const { data: hostsCount } = useQuery<
    IHostsCountResponse,
    Error,
    number,
    IHostsCountQueryKey[]
  >(
    [{ scope: "hosts_count", softwareVersionId: versionId }],
    ({ queryKey }) => hostsCountAPI.load(queryKey[0]),
    {
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
      select: (data) => data.count,
    }
  );

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderVulnTable = (swVersion: ISoftwareVersion) => {
    if (isIpadOrIphoneSoftwareSource(swVersion.source)) {
      const platformText = swVersion.source === "ios_apps" ? "iOS" : "iPadOS";
      return <VulnsNotSupported platformText={platformText} />;
    }
    return (
      <SoftwareVulnerabilitiesTable
        data={swVersion.vulnerabilities ?? []}
        itemName="software item"
        isLoading={isSoftwareVersionLoading}
        router={router}
        teamIdForApi={teamIdForApi}
      />
    );
  };

  const renderContent = () => {
    if (isSoftwareVersionLoading) {
      return <Spinner />;
    }

    if (!softwareVersion && !isSoftwareVersionError) {
      return null;
    }

    return (
      <>
        {isPremiumTier && (
          <TeamsHeader
            isOnGlobalTeam={isOnGlobalTeam}
            currentTeamId={currentTeamId}
            userTeams={userTeams}
            onTeamChange={onTeamChange}
          />
        )}
        {isSoftwareVersionError ? (
          <DetailsNoHosts
            header="Software not detected"
            details="No hosts have this software installed."
          />
        ) : (
          <>
            <SoftwareDetailsSummary
              title={`${softwareVersion.name}, ${softwareVersion.version}`}
              type={formatSoftwareType(softwareVersion)}
              hosts={hostsCount ?? 0}
              queryParams={{
                software_version_id: softwareVersion.id,
                team_id: teamIdForApi,
              }}
              name={softwareVersion.name}
              source={softwareVersion.source}
            />
            <Card
              borderRadiusSize="xxlarge"
              includeShadow
              className={`${baseClass}__vulnerabilities-section`}
            >
              <h2 className="section__header">Vulnerabilities</h2>
              {renderVulnTable(softwareVersion)}
            </Card>
          </>
        )}
      </>
    );
  };

  return (
    <MainContent className={baseClass}>
      <>{renderContent()}</>
    </MainContent>
  );
};
export default SoftwareVersionDetailsPage;
