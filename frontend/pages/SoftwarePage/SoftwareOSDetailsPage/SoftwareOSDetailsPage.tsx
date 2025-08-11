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
import { createMockLinuxOSVersion } from "__mocks__/operatingSystemsMock";
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

  const {
    data: { os_version: osVersionDetails2, counts_updated_at } = {},
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

  const osVersionDetails = createMockLinuxOSVersion();

  console.log("osversiondetails2", osVersionDetails2);
  console.log("osversiondetails", osVersionDetails);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderKernelsTable = () => {
    return (
      <OSKernelsTable
        data={osVersionDetails.kernels}
        isLoading={isLoading}
        router={router}
        teamIdForApi={teamIdForApi}
      />
    );
  };

  const renderVulnerabilitiesTable = () => {
    if (!osVersionDetails) {
      return null;
    }

    if (
      !VULN_SUPPORTED_PLATFORMS.includes(
        osVersionDetails.platform as Platform
      ) &&
      !isLinuxLike(osVersionDetails.platform) // 4.73 Linux vulns are now supported
    ) {
      const platformText =
        PLATFORM_DISPLAY_NAMES[osVersionDetails.platform] ||
        osVersionDetails.platform;
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

    const isLinuxPlatform = isLinuxLike(osVersionDetails.platform);
    const showKernelsCard = osVersionDetails.kernels.length > 0;

    // Vulns are associated with specific kernels hence hiding Vulns table on OS view
    // and showing vulns within OS > Kernels card
    const showVulnerabilitiesCard = !isLinuxPlatform;

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
            <Card
              borderRadiusSize="xxlarge"
              includeShadow
              className={`${baseClass}__summary-section`}
            >
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
            </Card>
            {showKernelsCard && (
              <Card
                borderRadiusSize="xxlarge"
                includeShadow
                className={`${baseClass}__summary-section`}
              >
                <CardHeader header="Kernels" />
                {renderKernelsTable()}
              </Card>
            )}
            {showVulnerabilitiesCard && (
              <Card
                borderRadiusSize="xxlarge"
                includeShadow
                className={`${baseClass}__vulnerabilities-section`}
              >
                <CardHeader header="Vulnerabilities" />
                {renderVulnerabilitiesTable()}
              </Card>
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
