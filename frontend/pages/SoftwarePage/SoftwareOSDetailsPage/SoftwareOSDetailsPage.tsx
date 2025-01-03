/** software/os/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";

import { ignoreAxiosError } from "interfaces/errors";
import {
  isLinuxLike,
  Platform,
  VULN_SUPPORTED_PLATFORMS,
} from "interfaces/platform";

import osVersionsAPI, {
  IOSVersionResponse,
  IGetOsVersionQueryKey,
} from "services/entities/operating_systems";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/SoftwareVulnerabilitiesTable";
import DetailsNoHosts from "../components/DetailsNoHosts";
import { VulnsNotSupported } from "../components/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";

const baseClass = "software-os-details-page";

interface ISoftwareOSDetailsRouteParams {
  id: string;
  team_id?: string;
}

type ISoftwareOSDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareOSDetailsRouteParams
>;

type QueryResult = {
  os_version: IOperatingSystemVersion;
  counts_updated_at?: string;
};

const SoftwareOSDetailsPage = ({
  routeParams,
  router,
  location,
}: ISoftwareOSDetailsPageProps) => {
  const { isPremiumTier, isOnGlobalTeam } = useContext(AppContext);
  const handlePageError = useErrorHandler();

  const osVersionIdFromURL = parseInt(routeParams.id, 10);

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
    data: { os_version: osVersionDetails, counts_updated_at } = {},
    isLoading,
    isError: isOsVersionError,
  } = useQuery<
    IOSVersionResponse,
    AxiosError,
    QueryResult,
    IGetOsVersionQueryKey[]
  >(
    [
      {
        scope: "osVersionDetails",
        os_version_id: osVersionIdFromURL,
        teamId: teamIdForApi,
      },
    ],
    ({ queryKey }) => osVersionsAPI.getOSVersion(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      enabled: !!osVersionIdFromURL,
      select: (data) => ({
        os_version: data.os_version,
        counts_updated_at: data.counts_updated_at,
      }),
      onError: (error) => {
        if (!ignoreAxiosError(error, [403, 404])) {
          handlePageError(error);
        }
      },
    }
  );

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderTable = () => {
    if (!osVersionDetails) {
      return null;
    }

    if (
      // TODO - detangle platform typing here
      !VULN_SUPPORTED_PLATFORMS.includes(osVersionDetails.platform as Platform)
    ) {
      const platformText = isLinuxLike(osVersionDetails.platform)
        ? "Linux"
        : PLATFORM_DISPLAY_NAMES[osVersionDetails.platform];
      return <VulnsNotSupported platformText={platformText} />;
    }

    return (
      <SoftwareVulnerabilitiesTable
        data={osVersionDetails.vulnerabilities}
        itemName="version"
        isLoading={isLoading}
        router={router}
        teamIdForApi={teamIdForApi}
      />
    );
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (!osVersionDetails && !isOsVersionError) {
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
        {isOsVersionError || !osVersionDetails ? (
          <DetailsNoHosts
            header="OS not detected"
            details="No hosts have this OS installed."
          />
        ) : (
          <>
            <SoftwareDetailsSummary
              title={osVersionDetails.name}
              hosts={osVersionDetails.hosts_count}
              countsUpdatedAt={counts_updated_at}
              queryParams={{
                os_name: osVersionDetails.name_only,
                os_version: osVersionDetails.version,
                team_id: teamIdForApi,
              }}
              name={osVersionDetails.platform}
            />
            <Card
              borderRadiusSize="xxlarge"
              includeShadow
              className={`${baseClass}__vulnerabilities-section`}
            >
              <h2>Vulnerabilities</h2>
              {renderTable()}
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

export default SoftwareOSDetailsPage;
