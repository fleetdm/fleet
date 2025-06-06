/** software/titles/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import paths from "router/paths";
import useTeamIdParam from "hooks/useTeamIdParam";
import { AppContext } from "context/app";
import { ignoreAxiosError } from "interfaces/errors";
import { ISoftwareTitleDetails } from "interfaces/software";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";

import { getPathWithQueryParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import DetailsNoHosts from "../components/cards/DetailsNoHosts";
import SoftwareSummaryCard from "./SoftwareSummaryCard";
import SoftwareInstallerCard from "./SoftwareInstallerCard";
import { getInstallerCardInfo } from "./helpers";

const baseClass = "software-title-details-page";

interface ISoftwareTitleDetailsRouteParams {
  id: string;
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
    config,
  } = useContext(AppContext);
  const handlePageError = useErrorHandler();

  // TODO: handle non integer values
  const softwareId = parseInt(routeParams.id, 10);
  const autoOpenGitOpsYamlModal =
    location.query.gitops_yaml === "true" && config?.gitops.gitops_mode_enabled;

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
    router.push(
      getPathWithQueryParams(paths.SOFTWARE_TITLES, {
        team_id: teamIdForApi,
      })
    );
  }, [refetchSoftwareTitle, router, softwareTitle, teamIdForApi]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderSoftwareInstallerCard = (title: ISoftwareTitleDetails) => {
    const hasPermission = Boolean(
      isOnGlobalTeam || isTeamAdmin || isTeamMaintainer || isTeamObserver
    );

    const showInstallerCard =
      currentTeamId !== APP_CONTEXT_ALL_TEAMS_ID &&
      hasPermission &&
      isAvailableForInstall;

    if (!showInstallerCard) {
      return null;
    }

    const {
      softwareTitleName,
      softwarePackage,
      name,
      version,
      addedTimestamp,
      status,
      isSelfService,
    } = getInstallerCardInfo(title);

    return (
      <SoftwareInstallerCard
        softwareTitleName={softwareTitleName}
        softwareInstaller={softwarePackage}
        name={name}
        version={version}
        addedTimestamp={addedTimestamp}
        status={status}
        isSelfService={isSelfService}
        softwareId={softwareId}
        teamId={currentTeamId ?? APP_CONTEXT_NO_TEAM_ID}
        teamIdForApi={teamIdForApi}
        onDelete={onDeleteInstaller}
        refetchSoftwareTitle={refetchSoftwareTitle}
        isLoading={isSoftwareTitleLoading}
        router={router}
        gitOpsYamlParam={autoOpenGitOpsYamlModal}
      />
    );
  };

  const renderSoftwareSummaryCard = (title: ISoftwareTitleDetails) => {
    return (
      <SoftwareSummaryCard
        title={title}
        softwareId={softwareId}
        teamId={teamIdForApi}
        isAvailableForInstall={isAvailableForInstall}
        isLoading={isSoftwareTitleLoading}
        router={router}
      />
    );
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
          {renderSoftwareSummaryCard(softwareTitle)}
          {renderSoftwareInstallerCard(softwareTitle)}
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
