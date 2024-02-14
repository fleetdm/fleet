/** software/titles/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";

import { ISoftwareTitle, formatSoftwareType } from "interfaces/software";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import Fleet404 from "pages/errors/Fleet404";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareTitleDetailsTable from "./SoftwareTitleDetailsTable";

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
  const { isPremiumTier, isOnGlobalTeam } = useContext(AppContext);

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
    includeNoTeam: false,
  });

  const {
    data: softwareTitle,
    isLoading: isSoftwareTitleLoading,
    isError: isSoftwareTitleError,
    error: softwareTitleError,
  } = useQuery<
    ISoftwareTitleResponse,
    AxiosError,
    ISoftwareTitle,
    IGetSoftwareTitleQueryKey[]
  >(
    [{ scope: "softwareById", softwareId, teamId: teamIdForApi }],
    ({ queryKey }) => softwareAPI.getSoftwareTitle(queryKey[0]),
    {
      select: (data) => data.software_title,
    }
  );

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderContent = () => {
    if (isSoftwareTitleLoading) {
      return <Spinner />;
    }

    if (isSoftwareTitleError) {
      // confirm okay to cast to AxiosError like this
      if (softwareTitleError.status === 404) {
        return <Fleet404 />;
      }
      return <TableDataError className={`${baseClass}__table-error`} />;
    }

    if (!softwareTitle) {
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
          title={softwareTitle.name}
          type={formatSoftwareType(softwareTitle)}
          versions={softwareTitle.versions.length}
          hosts={softwareTitle.hosts_count}
          queryParams={{ software_title_id: softwareId, team_id: teamIdForApi }}
          name={softwareTitle.name}
          source={softwareTitle.source}
        />
        <Card
          borderRadiusSize="large"
          includeShadow
          className={`${baseClass}__versions-section`}
        >
          <h2>Versions</h2>
          <SoftwareTitleDetailsTable
            router={router}
            data={softwareTitle.versions}
            isLoading={isSoftwareTitleLoading}
            teamIdForApi={teamIdForApi}
          />
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

export default SoftwareTitleDetailsPage;
