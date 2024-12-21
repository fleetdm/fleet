/** software/titles/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import paths from "router/paths";
import useTeamIdParam from "hooks/useTeamIdParam";
import { AppContext } from "context/app";
import {
  ISoftwareTitleDetails,
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import { ignoreAxiosError } from "interfaces/errors";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareTitleDetailsTable from "./SoftwareTitleDetailsTable";
import DetailsNoHosts from "../components/DetailsNoHosts";
import SoftwarePackageCard from "./SoftwarePackageCard";
import { getPackageCardInfo } from "./helpers";

const baseClass = "software-title-details-page";

interface ISoftwareTitleDetailsRouteParams {
  id: string;
  team_id?: string;
}

type ISoftwareTitleDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareTitleDetailsRouteParams
>;

const SoftwareTitleDetailsPage = ({
  router,
  routeParams,
  location,
}: ISoftwareTitleDetailsPageProps) => {
  const {
    isPremiumTier,
    isOnGlobalTeam,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamObserver,
  } = useContext(AppContext);
  const handlePageError = useErrorHandler();

  // TODO: handle non integer values
  const softwareId = parseInt(routeParams.id, 10);

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
    data: softwareTitle,
    isLoading: isSoftwareTitleLoading,
    isError: isSoftwareTitleError,
    refetch: refetchSoftwareTitle,
  } = useQuery<
    ISoftwareTitleResponse,
    AxiosError,
    ISoftwareTitleDetails,
    IGetSoftwareTitleQueryKey[]
  >(
    [{ scope: "softwareById", softwareId, teamId: teamIdForApi }],
    ({ queryKey }) => softwareAPI.getSoftwareTitle(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      select: (data) => data.software_title,
      onError: (error) => {
        if (!ignoreAxiosError(error, [403, 404])) {
          handlePageError(error);
        }
      },
    }
  );

  const isAvailableForInstall =
    !!softwareTitle?.software_package || !!softwareTitle?.app_store_app;

  const onDeleteInstaller = useCallback(() => {
    if (softwareTitle?.versions?.length) {
      refetchSoftwareTitle();
      return;
    }
    // redirect to software titles page if no versions are available
    if (teamIdForApi) {
      router.push(paths.SOFTWARE_TITLES.concat(`?team_id=${teamIdForApi}`));
    } else {
      router.push(paths.SOFTWARE_TITLES);
    }
  }, [refetchSoftwareTitle, router, softwareTitle, teamIdForApi]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderSoftwarePackageCard = (title: ISoftwareTitleDetails) => {
    const hasPermission = Boolean(
      isOnGlobalTeam || isTeamAdmin || isTeamMaintainer || isTeamObserver
    );

    const showPackageCard =
      currentTeamId !== APP_CONTEXT_ALL_TEAMS_ID &&
      hasPermission &&
      isAvailableForInstall;

    if (showPackageCard) {
      const packageCardData = getPackageCardInfo(title);
      return (
        <SoftwarePackageCard
          softwarePackage={packageCardData.softwarePackage}
          name={packageCardData.name}
          version={packageCardData.version}
          uploadedAt={packageCardData.uploadedAt}
          status={packageCardData.status}
          isSelfService={packageCardData.isSelfService}
          softwareId={softwareId}
          teamId={currentTeamId ?? APP_CONTEXT_NO_TEAM_ID}
          onDelete={onDeleteInstaller}
          refetchSoftwareTitle={refetchSoftwareTitle}
        />
      );
    }

    return null;
  };

  const renderContent = () => {
    if (isSoftwareTitleLoading) {
      return <Spinner />;
    }

    if (isSoftwareTitleError) {
      return (
        <DetailsNoHosts
          header="Software not detected"
          details="Expecting to see software? Check back later."
        />
      );
    }

    if (softwareTitle) {
      return (
        <>
          <SoftwareDetailsSummary
            title={softwareTitle.name}
            type={formatSoftwareType(softwareTitle)}
            versions={softwareTitle.versions?.length ?? 0}
            hosts={softwareTitle.hosts_count}
            queryParams={{
              software_title_id: softwareId,
              team_id: teamIdForApi,
            }}
            name={softwareTitle.name}
            source={softwareTitle.source}
            iconUrl={
              softwareTitle.app_store_app
                ? softwareTitle.app_store_app.icon_url
                : undefined
            }
          />
          {renderSoftwarePackageCard(softwareTitle)}
          <Card
            borderRadiusSize="xxlarge"
            includeShadow
            className={`${baseClass}__versions-section`}
          >
            <h2>Versions</h2>
            <SoftwareTitleDetailsTable
              router={router}
              data={softwareTitle.versions ?? []}
              isLoading={isSoftwareTitleLoading}
              teamIdForApi={teamIdForApi}
              isIPadOSOrIOSApp={isIpadOrIphoneSoftwareSource(
                softwareTitle.source
              )}
              isAvailableForInstall={isAvailableForInstall}
              countsUpdatedAt={softwareTitle.counts_updated_at}
            />
          </Card>
        </>
      );
    }

    return null;
  };

  return (
    <MainContent className={baseClass}>
      {isPremiumTier && (
        <TeamsHeader
          isOnGlobalTeam={isOnGlobalTeam}
          currentTeamId={currentTeamId}
          userTeams={userTeams}
          onTeamChange={onTeamChange}
        />
      )}
      <>{renderContent()}</>
    </MainContent>
  );
};

export default SoftwareTitleDetailsPage;
