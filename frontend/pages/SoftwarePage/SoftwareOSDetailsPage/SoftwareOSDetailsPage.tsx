/** software/os/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";

import osVersionsAPI, {
  IOSVersionResponse,
  IGetOsVersionQueryKey,
} from "services/entities/operating_systems";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { SUPPORT_LINK } from "utilities/constants";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import Fleet404 from "pages/errors/Fleet404";
import MainContent from "components/MainContent";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/SoftwareVulnerabilitiesTable";

const baseClass = "software-os-details-page";

interface INotSupportedVulnProps {
  platform: string;
}

const NotSupportedVuln = ({ platform }: INotSupportedVulnProps) => {
  return (
    <EmptyTable
      header="Vulnerabilities are not supported for this type of host"
      info={
        <>
          Interested in vulnerability management for{" "}
          {platform === "chrome" ? "Chromebooks" : "Linux hosts"}?{" "}
          <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
        </>
      }
    />
  );
};

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
  const { isPremiumTier, isOnGlobalTeam } = useContext(AppContext);

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
    includeNoTeam: false,
  });

  const {
    data: osVersionDetails,
    isLoading,
    isError: isOsVersionError,
    error: osVersionError,
  } = useQuery<
    IOSVersionResponse,
    AxiosError,
    IOperatingSystemVersion,
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
      enabled: !!osVersionIdFromURL,
      select: (data) => data.os_version,
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
      osVersionDetails.platform !== "darwin" &&
      osVersionDetails.platform !== "windows"
    ) {
      return <NotSupportedVuln platform={osVersionDetails.platform} />;
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

    if (isOsVersionError) {
      // confirm okay to cast to AxiosError like this
      if (osVersionError.status === 404) {
        return <Fleet404 />;
      }
      return <TableDataError className={`${baseClass}__table-error`} />;
    }

    if (!osVersionDetails) {
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
        <SoftwareDetailsSummary
          title={osVersionDetails.name}
          hosts={osVersionDetails.hosts_count}
          queryParams={{
            os_name: osVersionDetails.name_only,
            os_version: osVersionDetails.version,
            team_id: teamIdForApi,
          }}
          name={osVersionDetails.platform}
        />
        <Card
          borderRadiusSize="large"
          includeShadow
          className={`${baseClass}__vulnerabilities-section`}
        >
          <h2>Vulnerabilities</h2>
          {renderTable()}
        </Card>
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
