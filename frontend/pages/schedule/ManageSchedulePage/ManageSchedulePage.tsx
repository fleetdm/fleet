/* Conditionally renders global schedule and team schedules */

import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { ITeam } from "interfaces/team";
import { IQuery, IFleetQueriesResponse } from "interfaces/query";
import {
  IScheduledQuery,
  IEditScheduledQuery,
  ILoadAllGlobalScheduledQueriesResponse,
  IStoredScheduledQueriesResponse,
} from "interfaces/scheduled_query";
import paths from "router/paths";
import fleetQueriesAPI from "services/entities/queries";
import globalScheduledQueriesAPI from "services/entities/global_scheduled_queries";
import teamScheduledQueriesAPI from "services/entities/team_scheduled_queries";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import deepDifference from "utilities/deep_difference";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";

import ScheduleTable from "./components/ScheduleTable";
import ScheduleEditorModal from "./components/ScheduleEditorModal";
import RemoveScheduledQueryModal from "./components/RemoveScheduledQueryModal";

const baseClass = "manage-schedule-page";

const renderTable = (
  router: InjectedRouter,
  onRemoveScheduledQueryClick: (selectIds: number[]) => void,
  onEditScheduledQueryClick: (selectedQuery: IEditScheduledQuery) => void,
  onShowQueryClick: (selectedQuery: IEditScheduledQuery) => void,
  allScheduledQueriesList: IScheduledQuery[],
  allScheduledQueriesError: Error | null,
  toggleScheduleEditorModal: () => void,
  isOnGlobalTeam: boolean,
  selectedTeamData: ITeam | undefined,
  isLoadingGlobalScheduledQueries: boolean,
  isLoadingTeamScheduledQueries: boolean,
  errorQueries: Error | null
): JSX.Element => {
  return allScheduledQueriesError || errorQueries ? (
    <TableDataError />
  ) : (
    <ScheduleTable
      router={router}
      onRemoveScheduledQueryClick={onRemoveScheduledQueryClick}
      onEditScheduledQueryClick={onEditScheduledQueryClick}
      onShowQueryClick={onShowQueryClick}
      allScheduledQueriesList={allScheduledQueriesList}
      toggleScheduleEditorModal={toggleScheduleEditorModal}
      isOnGlobalTeam={isOnGlobalTeam}
      selectedTeamData={selectedTeamData}
      loadingInheritedQueriesTableData={isLoadingGlobalScheduledQueries}
      loadingTeamQueriesTableData={isLoadingTeamScheduledQueries}
    />
  );
};

const renderAllTeamsTable = (
  router: InjectedRouter,
  allTeamsScheduledQueriesList: IScheduledQuery[],
  allTeamsScheduledQueriesError: Error | null,
  isOnGlobalTeam: boolean,
  selectedTeamData: ITeam | undefined,
  isLoadingGlobalScheduledQueries: boolean,
  isLoadingTeamScheduledQueries: boolean
): JSX.Element => {
  return allTeamsScheduledQueriesError ? (
    <TableDataError />
  ) : (
    <div className={`${baseClass}__all-teams-table`}>
      <ScheduleTable
        router={router}
        inheritedQueries
        allScheduledQueriesList={allTeamsScheduledQueriesList}
        isOnGlobalTeam={isOnGlobalTeam}
        selectedTeamData={selectedTeamData}
        loadingInheritedQueriesTableData={isLoadingGlobalScheduledQueries}
        loadingTeamQueriesTableData={isLoadingTeamScheduledQueries}
      />
    </div>
  );
};

interface IFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  logging_type: string;
  platform: string;
  version: string;
  team_id?: number;
}

interface ITeamSchedulesPageProps {
  params: {
    team_id: string;
  };
  router: InjectedRouter; // v3
  route: any;
  location: any;
}

const ManageSchedulePage = ({
  router,
  location,
}: ITeamSchedulesPageProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { MANAGE_PACKS } = paths;
  const handleAdvanced = () => router.push(MANAGE_PACKS);

  const {
    isOnGlobalTeam,
    isPremiumTier,
    isFreeTier,
    isSandboxMode,
  } = useContext(AppContext);

  const {
    currentTeamId,
    isAnyTeamSelected,
    isRouteOk,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: false,
      observer_plus: false,
    },
  });

  const { data: teams, isLoading: isLoadingTeams } = useQuery<
    ILoadTeamsResponse,
    Error,
    ITeam[]
  >(["teams"], () => teamsAPI.loadAll(), {
    enabled: isRouteOk && !!isPremiumTier,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    select: (data) => data.teams,
  });

  const {
    data: fleetQueries,
    isLoading: isLoadingFleetQueries,
    error: errorQueries,
  } = useQuery<IFleetQueriesResponse, Error, IQuery[]>(
    ["fleetQueries"],
    () => fleetQueriesAPI.loadAll(),
    {
      enabled: isRouteOk,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
      select: (data) => data.queries,
    }
  );

  const {
    data: globalScheduledQueries,
    error: globalScheduledQueriesError,
    isLoading: isLoadingGlobalScheduledQueries,
    refetch: refetchGlobalScheduledQueries,
  } = useQuery<
    ILoadAllGlobalScheduledQueriesResponse,
    Error,
    IScheduledQuery[]
  >(["globalScheduledQueries"], () => globalScheduledQueriesAPI.loadAll(), {
    enabled: isRouteOk,
    select: (data) => data.global_schedule,
  });

  const {
    data: teamScheduledQueries,
    error: teamScheduledQueriesError,
    isLoading: isLoadingTeamScheduledQueries,
    refetch: refetchTeamScheduledQueries,
  } = useQuery<IStoredScheduledQueriesResponse, Error, IScheduledQuery[]>(
    ["teamScheduledQueries", teamIdForApi],
    () => teamScheduledQueriesAPI.loadAll(teamIdForApi),
    {
      enabled: isRouteOk && isPremiumTier && !!teamIdForApi,
      select: (data) => data.scheduled,
    }
  );

  const refetchScheduledQueries = useCallback(() => {
    refetchGlobalScheduledQueries();
    if (isAnyTeamSelected) {
      refetchTeamScheduledQueries();
    }
  }, [
    isAnyTeamSelected,
    refetchGlobalScheduledQueries,
    refetchTeamScheduledQueries,
  ]);

  const allScheduledQueriesList =
    (isAnyTeamSelected ? teamScheduledQueries : globalScheduledQueries) || [];
  const allScheduledQueriesError = isAnyTeamSelected
    ? teamScheduledQueriesError
    : globalScheduledQueriesError;

  const inheritedScheduledQueriesList = globalScheduledQueries;
  const inheritedScheduledQueriesError = globalScheduledQueriesError;

  const inheritedQueryOrQueries =
    inheritedScheduledQueriesList?.length === 1 ? "query" : "queries";

  const selectedTeamData = isAnyTeamSelected
    ? teams?.find((team: ITeam) => teamIdForApi === team.id)
    : undefined;

  const [isUpdatingScheduledQuery, setIsUpdatingScheduledQuery] = useState(
    false
  );
  const [showInheritedQueries, setShowInheritedQueries] = useState(false);
  const [showScheduleEditorModal, setShowScheduleEditorModal] = useState(false);
  const [showShowQueryModal, setShowShowQueryModal] = useState(false);
  const [showPreviewDataModal, setShowPreviewDataModal] = useState(false);
  const [
    showRemoveScheduledQueryModal,
    setShowRemoveScheduledQueryModal,
  ] = useState(false);
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[] | never[]>(
    []
  );
  const [
    selectedScheduledQuery,
    setSelectedScheduledQuery,
  ] = useState<IEditScheduledQuery>();

  const toggleInheritedQueries = () => {
    setShowInheritedQueries(!showInheritedQueries);
  };

  const togglePreviewDataModal = useCallback(() => {
    setShowPreviewDataModal(!showPreviewDataModal);
  }, [setShowPreviewDataModal, showPreviewDataModal]);

  const toggleScheduleEditorModal = useCallback(() => {
    setSelectedScheduledQuery(undefined); // create modal renders
    setShowScheduleEditorModal(!showScheduleEditorModal);
  }, [showScheduleEditorModal, setShowScheduleEditorModal]);

  const toggleShowQueryModal = useCallback(() => {
    setSelectedScheduledQuery(undefined);
    setShowShowQueryModal(!showShowQueryModal);
  }, [showShowQueryModal, setShowShowQueryModal]);

  const toggleRemoveScheduledQueryModal = useCallback(() => {
    setShowRemoveScheduledQueryModal(!showRemoveScheduledQueryModal);
  }, [showRemoveScheduledQueryModal, setShowRemoveScheduledQueryModal]);

  const onRemoveScheduledQueryClick = (
    selectedTableQueryIds: number[]
  ): void => {
    toggleRemoveScheduledQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onShowQueryClick = (selectedQuery: IEditScheduledQuery): void => {
    toggleShowQueryModal();
    setSelectedScheduledQuery(selectedQuery);
  };

  const onEditScheduledQueryClick = (
    selectedQuery: IEditScheduledQuery
  ): void => {
    toggleScheduleEditorModal();
    setSelectedScheduledQuery(selectedQuery); // edit modal renders
  };

  const onRemoveScheduledQuerySubmit = useCallback(() => {
    setIsUpdatingScheduledQuery(true);
    const promises = selectedQueryIds.map((id: number) => {
      return isAnyTeamSelected
        ? teamScheduledQueriesAPI.destroy(teamIdForApi, id)
        : globalScheduledQueriesAPI.destroy({ id });
    });
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";
    return Promise.all(promises)
      .then(() => {
        renderFlash(
          "success",
          `Successfully removed scheduled ${queryOrQueries}.`
        );
        toggleRemoveScheduledQueryModal();
        refetchScheduledQueries();
      })
      .catch(() => {
        renderFlash(
          "error",
          `Unable to remove scheduled ${queryOrQueries}. Please try again.`
        );
        toggleRemoveScheduledQueryModal();
      })
      .finally(() => {
        refetchGlobalScheduledQueries();
        setIsUpdatingScheduledQuery(false);
      });
  }, [
    selectedQueryIds,
    isAnyTeamSelected,
    teamIdForApi,
    renderFlash,
    toggleRemoveScheduledQueryModal,
    refetchScheduledQueries,
    refetchGlobalScheduledQueries,
  ]);

  const onAddScheduledQuerySubmit = useCallback(
    (formData: IFormData, editQuery: IEditScheduledQuery | undefined) => {
      setIsUpdatingScheduledQuery(true);
      if (editQuery) {
        const updatedAttributes = deepDifference(formData, editQuery);

        const editResponse =
          editQuery.type === "team_scheduled_query"
            ? teamScheduledQueriesAPI.update(editQuery, updatedAttributes)
            : globalScheduledQueriesAPI.update(editQuery, updatedAttributes);

        editResponse
          .then(() => {
            renderFlash(
              "success",
              `Successfully updated ${formData.name} in the schedule.`
            );
            refetchScheduledQueries();
            toggleScheduleEditorModal();
          })
          .catch(() => {
            renderFlash(
              "error",
              "Could not update scheduled query. Please try again."
            );
          })
          .finally(() => {
            setIsUpdatingScheduledQuery(false);
            refetchGlobalScheduledQueries();
          });
      } else {
        const createResponse = isAnyTeamSelected
          ? teamScheduledQueriesAPI.create({ ...formData })
          : globalScheduledQueriesAPI.create({ ...formData });

        createResponse
          .then(() => {
            renderFlash(
              "success",
              `Successfully added ${formData.name} to the schedule.`
            );
            refetchScheduledQueries();
            toggleScheduleEditorModal();
          })
          .catch(() => {
            renderFlash("error", "Could not schedule query. Please try again.");
          })
          .finally(() => {
            setIsUpdatingScheduledQuery(false);
            refetchGlobalScheduledQueries();
          });
      }
    },
    [
      isAnyTeamSelected,
      refetchGlobalScheduledQueries,
      refetchScheduledQueries,
      renderFlash,
      toggleScheduleEditorModal,
    ]
  );

  if (!isRouteOk || (isPremiumTier && !userTeams?.length)) {
    return (
      <div className={`${baseClass}__loading-spinner`}>
        <Spinner />
      </div>
    );
  }

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Schedule</h1>}
                {isPremiumTier &&
                  userTeams &&
                  (userTeams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      selectedTeamId={currentTeamId}
                      currentUserTeams={userTeams || []}
                      onChange={handleTeamChange}
                      isSandboxMode={isSandboxMode}
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  userTeams &&
                  userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
              </div>
            </div>
          </div>
          {allScheduledQueriesList?.length !== 0 && !allScheduledQueriesError && (
            <div className={`${baseClass}__action-button-container`}>
              {/* NOTE:  Product decision to remove packs from UI
              {isOnGlobalTeam && (
                <Button
                  variant="inverse"
                  onClick={handleAdvanced}
                  className={`${baseClass}__advanced-button`}
                >
                  Advanced
                </Button>
              )} */}
              <Button
                variant="brand"
                className={`${baseClass}__schedule-button`}
                onClick={toggleScheduleEditorModal}
              >
                Schedule a query
              </Button>
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          {!isLoadingTeams && (
            <div>
              {isAnyTeamSelected ? (
                <p>
                  Schedule queries for{" "}
                  <strong>all hosts assigned to this team</strong>
                </p>
              ) : (
                <p>
                  Schedule queries to run at regular intervals across{" "}
                  <strong>all of your hosts</strong>
                </p>
              )}
            </div>
          )}
        </div>
        <div>
          {isLoadingTeams ||
          isLoadingFleetQueries ||
          isLoadingGlobalScheduledQueries ||
          isLoadingTeamScheduledQueries ? (
            <Spinner />
          ) : (
            renderTable(
              router,
              onRemoveScheduledQueryClick,
              onEditScheduledQueryClick,
              onShowQueryClick,
              allScheduledQueriesList,
              allScheduledQueriesError,
              toggleScheduleEditorModal,
              isOnGlobalTeam || false,
              selectedTeamData,
              isLoadingGlobalScheduledQueries,
              isLoadingTeamScheduledQueries,
              errorQueries
            )
          )}
        </div>
        {/* must use ternary for NaN */}
        {isAnyTeamSelected &&
        inheritedScheduledQueriesList &&
        inheritedScheduledQueriesList.length > 0 ? (
          <RevealButton
            isShowing={showInheritedQueries}
            className={baseClass}
            hideText={`Hide ${inheritedScheduledQueriesList.length} inherited ${inheritedQueryOrQueries}`}
            showText={`Show ${inheritedScheduledQueriesList.length} inherited ${inheritedQueryOrQueries}`}
            caretPosition={"before"}
            tooltipHtml={
              'Queries from the "All teams"<br/>schedule run on this team’s hosts.'
            }
            onClick={toggleInheritedQueries}
          />
        ) : null}
        {showInheritedQueries &&
          inheritedScheduledQueriesList &&
          renderAllTeamsTable(
            router,
            inheritedScheduledQueriesList,
            inheritedScheduledQueriesError,
            isOnGlobalTeam || false,
            selectedTeamData,
            isLoadingGlobalScheduledQueries,
            isLoadingTeamScheduledQueries
          )}
        {showScheduleEditorModal && fleetQueries && (
          <ScheduleEditorModal
            onClose={toggleScheduleEditorModal}
            onScheduleSubmit={onAddScheduledQuerySubmit}
            allQueries={fleetQueries}
            editQuery={selectedScheduledQuery}
            teamId={teamIdForApi}
            togglePreviewDataModal={togglePreviewDataModal}
            showPreviewDataModal={showPreviewDataModal}
            isUpdatingScheduledQuery={isUpdatingScheduledQuery}
          />
        )}
        {showRemoveScheduledQueryModal && (
          <RemoveScheduledQueryModal
            onCancel={toggleRemoveScheduledQueryModal}
            onSubmit={onRemoveScheduledQuerySubmit}
            isUpdatingScheduledQuery={isUpdatingScheduledQuery}
          />
        )}
        {showShowQueryModal && (
          <ShowQueryModal
            query={selectedScheduledQuery?.query}
            onCancel={toggleShowQueryModal}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManageSchedulePage;
