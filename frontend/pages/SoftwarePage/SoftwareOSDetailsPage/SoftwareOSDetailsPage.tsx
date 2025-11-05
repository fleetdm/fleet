/** software/os/:id */

import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";

import { ignoreAxiosError } from "interfaces/errors";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import {
  isLinuxLike,
  Platform,
  VULN_SUPPORTED_PLATFORMS,
} from "interfaces/platform";

import osVersionsAPI, {
  IOSVersionResponse,
  IGetOsVersionQueryKey,
} from "services/entities/operating_systems";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";
import CardHeader from "components/CardHeader";

import SoftwareDetailsSummary from "../components/cards/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/tables/SoftwareVulnerabilitiesTable";
import DetailsNoHosts from "../components/cards/DetailsNoHosts";
import { VulnsNotSupported } from "../components/tables/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";
import OSKernelsTable from "../components/tables/OSKernelsTable";

const baseClass = "software-os-details-page";

interface ISummaryCardProps {
  osVersion: IOperatingSystemVersion;
  countsUpdatedAt: string | undefined;
  teamIdForApi?: number;
}

export const SummaryCard = ({
  osVersion,
  countsUpdatedAt,
  teamIdForApi,
}: ISummaryCardProps) => (
  <Card borderRadiusSize="xxlarge" className={`${baseClass}__summary-section`}>
    <SoftwareDetailsSummary
      title={osVersion.name}
      hosts={osVersion.hosts_count}
      countsUpdatedAt={countsUpdatedAt}
      queryParams={{
        os_name: osVersion.name_only,
        os_version: osVersion.version,
        team_id: teamIdForApi,
      }}
      name={osVersion.platform}
      isOperatingSystem
    />
  </Card>
);

interface IVulnerabilitiesCardProps {
  osVersion: IOperatingSystemVersion;
  isLoading: boolean;
  router: InjectedRouter;
  teamIdForApi?: number;
}

export const VulnerabilitiesCard = ({
  osVersion,
  isLoading,
  router,
  teamIdForApi,
}: IVulnerabilitiesCardProps) => {
  const supportsVulns =
    VULN_SUPPORTED_PLATFORMS.includes(osVersion.platform as Platform) ||
    isLinuxLike(osVersion.platform);

  return (
    <Card
      borderRadiusSize="xxlarge"
      className={`${baseClass}__vulnerabilities-section`}
    >
      <CardHeader header="Vulnerabilities" />
      {supportsVulns ? (
        <SoftwareVulnerabilitiesTable
          data={osVersion.vulnerabilities}
          itemName="version"
          isLoading={isLoading}
          router={router}
          teamIdForApi={teamIdForApi}
        />
      ) : (
        <VulnsNotSupported
          platformText={
            PLATFORM_DISPLAY_NAMES[osVersion.platform] || osVersion.platform
          }
        />
      )}
    </Card>
  );
};

interface IKernelsCardProps {
  osVersion: IOperatingSystemVersion;
  isLoading: boolean;
  router: InjectedRouter;
  teamIdForApi?: number;
}

export const KernelsCard = ({
  osVersion,
  isLoading,
  router,
  teamIdForApi,
}: IKernelsCardProps) => (
  <Card borderRadiusSize="xxlarge" className={`${baseClass}__summary-section`}>
    <CardHeader header="Kernels" />
    <OSKernelsTable
      osName={osVersion.name_only}
      osVersion={osVersion.version}
      data={osVersion.kernels}
      isLoading={isLoading}
      router={router}
      teamIdForApi={teamIdForApi}
    />
  </Card>
);

interface ISoftwareOSDetailsRouteParams {
  id: string;
  team_id?: string;
}

type ISoftwareOSDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareOSDetailsRouteParams
>;

const SoftwareOSDetailsPage = ({
  routeParams,
  router,
  location,
}: ISoftwareOSDetailsPageProps) => {
  const { isPremiumTier, isOnGlobalTeam, config } = useContext(AppContext);
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

  // Track whether we need to fetch all vulnerabilities
  const [maxVulnerabilities, setMaxVulnerabilities] = useState<
    number | undefined
  >(0);

  const {
    data: { os_version: osVersionDetails, counts_updated_at } = {},
    isLoading,
    isError: isOsVersionError,
  } = useQuery<
    IOSVersionResponse,
    AxiosError,
    IOSVersionResponse,
    IGetOsVersionQueryKey[]
  >(
    [
      {
        scope: "osVersionDetails",
        os_version_id: osVersionIdFromURL,
        teamId: teamIdForApi,
        max_vulnerabilities: maxVulnerabilities,
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
      onSuccess: (data) => {
        const {
          os_version: { platform, vulnerabilities_count },
        } = data;
        if (
          !isLinuxLike(platform) &&
          vulnerabilities_count &&
          vulnerabilities_count > 0 &&
          maxVulnerabilities === 0
        ) {
          setMaxVulnerabilities(undefined);
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

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (!osVersionDetails && !isOsVersionError) {
      return null;
    }

    // Linux vulns are associated with specific kernels hence design
    // hiding default vulns table and showing vulns within OS > Kernels card
    const isLinuxPlatform = isLinuxLike(osVersionDetails?.platform || "");

    return (
      <>
        {isPremiumTier && !config?.partnerships?.enable_primo && (
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
            <SummaryCard
              osVersion={osVersionDetails}
              countsUpdatedAt={counts_updated_at}
              teamIdForApi={teamIdForApi}
            />
            {!isLinuxPlatform && (
              <VulnerabilitiesCard
                osVersion={osVersionDetails}
                isLoading={isLoading}
                router={router}
                teamIdForApi={teamIdForApi}
              />
            )}
            {isLinuxPlatform && (
              <KernelsCard
                osVersion={osVersionDetails}
                isLoading={isLoading}
                router={router}
                teamIdForApi={teamIdForApi}
              />
            )}
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
