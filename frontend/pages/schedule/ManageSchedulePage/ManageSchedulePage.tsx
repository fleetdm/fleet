/* Conditionally renders global schedule and team schedules */

import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";
import { AppContext } from "context/app";
import { InjectedRouter } from "react-router/lib/Router";
import { find } from "lodash";

// @ts-ignore
import deepDifference from "utilities/deep_difference";
import { ITeam } from "interfaces/team";
import {
  IGlobalScheduledQuery,
  ILoadAllGlobalScheduledQueriesResponse,
} from "interfaces/global_scheduled_query";
import {
  ITeamScheduledQuery,
  ILoadAllTeamScheduledQueriesResponse,
} from "interfaces/team_scheduled_query";
import fleetQueriesAPI from "services/entities/queries";
import globalScheduledQueriesAPI from "services/entities/global_scheduled_queries";
import teamScheduledQueriesAPI from "services/entities/team_scheduled_queries";
import teamsAPI from "services/entities/teams";
import usersAPI, { IGetMeResponse } from "services/entities/users";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import sortUtils from "utilities/sort";
import paths from "router/paths";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import IconToolTip from "components/IconToolTip";
import TableDataError from "components/TableDataError";
import TooltipWrapper from "components/TooltipWrapper";
import ScheduleListWrapper from "./components/ScheduleListWrapper";
import ScheduleEditorModal from "./components/ScheduleEditorModal";
import RemoveScheduledQueryModal from "./components/RemoveScheduledQueryModal";

const baseClass = "manage-schedule-page";

const renderTable = (
  onRemoveScheduledQueryClick: (selectIds: number[]) => void,
  onEditScheduledQueryClick: (
    selectedQuery: IGlobalScheduledQuery | ITeamScheduledQuery
  ) => void,
  allScheduledQueriesList: IGlobalScheduledQuery[] | ITeamScheduledQuery[],
  allScheduledQueriesError: Error | null,
  toggleScheduleEditorModal: () => void,
  isOnGlobalTeam: boolean,
  selectedTeamData: ITeam | undefined,
  isLoadingGlobalScheduledQueries: boolean,
  isLoadingTeamScheduledQueries: boolean
): JSX.Element => {
  return allScheduledQueriesError ? (
    <TableDataError />
  ) : (
    <ScheduleListWrapper
      onRemoveScheduledQueryClick={onRemoveScheduledQueryClick}
      onEditScheduledQueryClick={onEditScheduledQueryClick}
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
  allTeamsScheduledQueriesList: IGlobalScheduledQuery[],
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
      <ScheduleListWrapper
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

interface ITeamSchedulesPageProps {
  params: {
    team_id: string;
  };
  router: InjectedRouter; // v3
}
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

const ManageSchedulePage = ({
  params: { team_id },
  router,
}: ITeamSchedulesPageProps): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS, MANAGE_SCHEDULE, MANAGE_TEAM_SCHEDULE } = paths;
  const handleAdvanced = () => router.push(MANAGE_PACKS);

  const {
    availableTeams,
    currentUser,
    isOnGlobalTeam,
    isPremiumTier,
    isFreeTier,
    currentTeam,
    setAvailableTeams,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);

  const teamId = parseInt(team_id, 10) || 0;

  const filterAndSortTeamOptions = (allTeams: ITeam[], userTeams: ITeam[]) => {
    const filteredSortedTeams = allTeams
      .sort((teamA: ITeam, teamB: ITeam) =>
        sortUtils.caseInsensitiveAsc(teamA.name, teamB.name)
      )
      .filter((team: ITeam) => {
        const userTeam = userTeams.find(
          (thisUserTeam) => thisUserTeam.id === team.id
        );
        return userTeam?.role !== "observer" ? team : null;
      });

    return filteredSortedTeams;
  };

  useQuery(["me"], () => usersAPI.me(), {
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(available_teams);
    },
  });

  const { data: teams, isLoading: isLoadingTeams } = useQuery(
    ["teams"],
    () => teamsAPI.loadAll({}),
    {
      enabled: !!isPremiumTier,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
      select: (data) => {
        return currentUser?.teams
          ? filterAndSortTeamOptions(data.teams, currentUser.teams)
          : data.teams;
      },
    }
  );

  const { data: fleetQueries, isLoading: isLoadingFleetQueries } = useQuery(
    ["fleetQueries"],
    () => fleetQueriesAPI.loadAll(),
    {
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
    IGlobalScheduledQuery[]
  >(["globalScheduledQueries"], () => globalScheduledQueriesAPI.loadAll(), {
    enabled: !!availableTeams,
    select: (data) => data.global_schedule,
  });

  let selectedTeamId = currentTeam?.id ? currentTeam.id : teamId || 0;

  // No access for observers of currentTeam, shown first team with RBAC
  if (selectedTeamId) {
    const selectedTeam = currentUser?.teams.find(
      (team) => team.id === selectedTeamId
    );
    if (selectedTeam?.role === "observer") {
      const teamWithAccess = currentUser?.teams.find(
        (team) => team.role !== "observer"
      );
      if (teamWithAccess) {
        selectedTeamId = teamWithAccess?.id;
      }
    }
  }

  const {
    data: teamScheduledQueries,
    error: teamScheduledQueriesError,
    isLoading: isLoadingTeamScheduledQueries,
    refetch: refetchTeamScheduledQueries,
  } = useQuery<
    ILoadAllTeamScheduledQueriesResponse,
    Error,
    ITeamScheduledQuery[]
  >(
    ["teamScheduledQueries", selectedTeamId],
    () => teamScheduledQueriesAPI.loadAll(selectedTeamId),
    {
      enabled: !!availableTeams && isPremiumTier && !!selectedTeamId,
      select: (data) => data.scheduled,
    }
  );

  const refetchScheduledQueries = () => {
    refetchGlobalScheduledQueries();
    if (selectedTeamId !== 0) {
      refetchTeamScheduledQueries();
    }
  };

  const findAvailableTeam = (id: number) => {
    return availableTeams?.find((t) => t.id === id);
  };

  const handleTeamSelect = (id: number) => {
    if (id) {
      router.push(MANAGE_TEAM_SCHEDULE(id));
    } else {
      router.push(MANAGE_SCHEDULE);
    }
    const selectedTeam = find(teams, ["id", id]);
    setCurrentTeam(selectedTeam);
  };

  if (!isOnGlobalTeam && !selectedTeamId && teams) {
    handleTeamSelect(teams[0].id);
  }

  // If team_id from URL query params is not valid, we instead use a default team
  // either the current team (if any) or all teams (for global users) or
  // the first available team (for non-global users)
  const getValidatedTeamId = () => {
    if (findAvailableTeam(selectedTeamId)) {
      return selectedTeamId;
    }
    if (!selectedTeamId && currentTeam) {
      return currentTeam.id;
    }
    if (!selectedTeamId && !currentTeam && !isOnGlobalTeam && availableTeams) {
      return availableTeams[0]?.id;
    }
    return 0;
  };

  // If team_id or currentTeam doesn't match validated id, switch to validated id
  useEffect(() => {
    if (availableTeams) {
      const validatedId = getValidatedTeamId();

      if (validatedId !== currentTeam?.id || validatedId !== selectedTeamId) {
        handleTeamSelect(validatedId);
      }
    }
  }, [availableTeams]);

  const allScheduledQueriesList =
    (selectedTeamId ? teamScheduledQueries : globalScheduledQueries) || [];
  const allScheduledQueriesError = selectedTeamId
    ? teamScheduledQueriesError
    : globalScheduledQueriesError;

  const inheritedScheduledQueriesList = globalScheduledQueries;
  const inheritedScheduledQueriesError = globalScheduledQueriesError;

  const inheritedQueryOrQueries =
    inheritedScheduledQueriesList?.length === 1 ? "query" : "queries";

  const selectedTeam = !selectedTeamId ? "global" : selectedTeamId;

  const selectedTeamData =
    teams?.find((team: ITeam) => selectedTeam === team.id) || undefined;

  const [showInheritedQueries, setShowInheritedQueries] = useState<boolean>(
    false
  );
  const [showScheduleEditorModal, setShowScheduleEditorModal] = useState(false);
  const [showPreviewDataModal, setShowPreviewDataModal] = useState(false);
  const [
    showRemoveScheduledQueryModal,
    setShowRemoveScheduledQueryModal,
  ] = useState(false);
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[] | never[]>(
    []
  );
  const [selectedScheduledQuery, setSelectedScheduledQuery] = useState<
    IGlobalScheduledQuery | ITeamScheduledQuery
  >();

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

  const toggleRemoveScheduledQueryModal = useCallback(() => {
    setShowRemoveScheduledQueryModal(!showRemoveScheduledQueryModal);
  }, [showRemoveScheduledQueryModal, setShowRemoveScheduledQueryModal]);

  const onRemoveScheduledQueryClick = (
    selectedTableQueryIds: number[]
  ): void => {
    toggleRemoveScheduledQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onEditScheduledQueryClick = (
    selectedQuery: IGlobalScheduledQuery | ITeamScheduledQuery
  ): void => {
    toggleScheduleEditorModal();
    setSelectedScheduledQuery(selectedQuery); // edit modal renders
  };

  const onRemoveScheduledQuerySubmit = useCallback(() => {
    const promises = selectedQueryIds.map((id: number) => {
      return selectedTeamId
        ? teamScheduledQueriesAPI.destroy(selectedTeamId, id)
        : globalScheduledQueriesAPI.destroy({ id });
    });
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";
    return Promise.all(promises)
      .then(() => {
        dispatch(
          renderFlash(
            "success",
            `Successfully removed scheduled ${queryOrQueries}.`
          )
        );
        toggleRemoveScheduledQueryModal();
        refetchScheduledQueries();
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove scheduled ${queryOrQueries}. Please try again.`
          )
        );
        toggleRemoveScheduledQueryModal();
      });
  }, [
    dispatch,
    selectedTeamId,
    selectedQueryIds,
    toggleRemoveScheduledQueryModal,
    refetchScheduledQueries,
  ]);

  const onAddScheduledQuerySubmit = useCallback(
    (
      formData: IFormData,
      editQuery: IGlobalScheduledQuery | ITeamScheduledQuery | undefined
    ) => {
      if (editQuery) {
        const updatedAttributes = deepDifference(formData, editQuery);

        const editResponse = selectedTeamId
          ? teamScheduledQueriesAPI.update(editQuery, updatedAttributes)
          : globalScheduledQueriesAPI.update(editQuery, updatedAttributes);

        editResponse
          .then(() => {
            dispatch(
              renderFlash(
                "success",
                `Successfully updated ${formData.name} in the schedule.`
              )
            );
            refetchScheduledQueries();
          })
          .catch(() => {
            dispatch(
              renderFlash(
                "error",
                "Could not update scheduled query. Please try again."
              )
            );
          });
      } else {
        const createResponse = selectedTeamId
          ? teamScheduledQueriesAPI.create({ ...formData })
          : globalScheduledQueriesAPI.create({ ...formData });

        createResponse
          .then(() => {
            dispatch(
              renderFlash(
                "success",
                `Successfully added ${formData.name} to the schedule.`
              )
            );
            refetchScheduledQueries();
          })
          .catch(() => {
            dispatch(
              renderFlash(
                "error",
                "Could not schedule query. Please try again."
              )
            );
          });
      }
      toggleScheduleEditorModal();
    },
    [dispatch, selectedTeamId, toggleScheduleEditorModal]
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Schedule</h1>}
                {isPremiumTier &&
                  teams &&
                  (teams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      selectedTeamId={selectedTeamId}
                      currentUserTeams={teams || []}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  teams &&
                  teams.length === 1 && <h1>{teams[0].name}</h1>}
              </div>
            </div>
          </div>
          {allScheduledQueriesList?.length !== 0 && !allScheduledQueriesError && (
            <div className={`${baseClass}__action-button-container`}>
              {isOnGlobalTeam && (
                <Button
                  variant="inverse"
                  onClick={handleAdvanced}
                  className={`${baseClass}__advanced-button`}
                >
                  Advanced
                </Button>
              )}
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
              {selectedTeamId ? (
                <p>
                  Schedule queries for{" "}
                  <strong>all hosts assigned to this team.</strong>
                </p>
              ) : (
                <p>
                  Schedule queries to run at regular intervals across{" "}
                  <strong>all of your hosts.</strong>
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
              onRemoveScheduledQueryClick,
              onEditScheduledQueryClick,
              allScheduledQueriesList,
              allScheduledQueriesError,
              toggleScheduleEditorModal,
              isOnGlobalTeam || false,
              selectedTeamData,
              isLoadingGlobalScheduledQueries,
              isLoadingTeamScheduledQueries
            )
          )}
        </div>
        {/* must use ternary for NaN */}
        {selectedTeamId &&
        inheritedScheduledQueriesList &&
        inheritedScheduledQueriesList.length > 0 ? (
          <span>
            <Button
              variant="unstyled"
              className={`${showInheritedQueries ? "upcarat" : "rightcarat"} 
                    ${baseClass}__inherited-queries-button`}
              onClick={toggleInheritedQueries}
              >
                <TooltipWrapper tipContent={'Queries from the "All teams"<br/>schedule run on this team’s hosts.'}>
                  {showInheritedQueries
                    ? `Hide ${inheritedScheduledQueriesList.length} inherited ${inheritedQueryOrQueries}`
                    : `Show ${inheritedScheduledQueriesList.length} inherited ${inheritedQueryOrQueries}`}
                </TooltipWrapper>
            </Button>
          </span>
        ) : null}
        {showInheritedQueries &&
          inheritedScheduledQueriesList &&
          renderAllTeamsTable(
            inheritedScheduledQueriesList,
            inheritedScheduledQueriesError,
            isOnGlobalTeam || false,
            selectedTeamData,
            isLoadingGlobalScheduledQueries,
            isLoadingTeamScheduledQueries
          )}
        {showScheduleEditorModal && (
          <ScheduleEditorModal
            onCancel={toggleScheduleEditorModal}
            onScheduleSubmit={onAddScheduledQuerySubmit}
            allQueries={fleetQueries}
            editQuery={selectedScheduledQuery}
            teamId={selectedTeamId}
            togglePreviewDataModal={togglePreviewDataModal}
            showPreviewDataModal={showPreviewDataModal}
          />
        )}
        {showRemoveScheduledQueryModal && (
          <RemoveScheduledQueryModal
            onCancel={toggleRemoveScheduledQueryModal}
            onSubmit={onRemoveScheduledQuerySubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManageSchedulePage;
