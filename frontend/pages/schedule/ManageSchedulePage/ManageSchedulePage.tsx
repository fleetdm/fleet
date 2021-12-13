/* Conditionally renders global schedule and team schedules */

import React, { useState, useCallback, useEffect, useContext } from "react";
import { useQuery } from "react-query";
import { useDispatch, useSelector } from "react-redux";
import { AppContext } from "context/app";
import { push } from "react-router-redux";
import { find } from "lodash";

// @ts-ignore
import deepDifference from "utilities/deep_difference";
import { ITeam } from "interfaces/team";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import { ITeamScheduledQuery } from "interfaces/team_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";
// @ts-ignore
import teamScheduledQueryActions from "redux/nodes/entities/team_scheduled_queries/actions";
import fleetQueriesAPI from "services/entities/queries";
import teamsAPI from "services/entities/teams";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import sortUtils from "utilities/sort";

import paths from "router/paths";
import Button from "components/buttons/Button";
// @ts-ignore
import TeamsDropdown from "components/TeamsDropdown";
import IconToolTip from "components/IconToolTip";
import TableDataError from "components/TableDataError";
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
  allScheduledQueriesError: { name: string; reason: string }[],
  toggleScheduleEditorModal: () => void,
  isOnGlobalTeam: boolean,
  selectedTeamData: ITeam | undefined
): JSX.Element => {
  if (Object.keys(allScheduledQueriesError).length !== 0) {
    return <TableDataError />;
  }

  return (
    <ScheduleListWrapper
      onRemoveScheduledQueryClick={onRemoveScheduledQueryClick}
      onEditScheduledQueryClick={onEditScheduledQueryClick}
      allScheduledQueriesList={allScheduledQueriesList}
      toggleScheduleEditorModal={toggleScheduleEditorModal}
      isOnGlobalTeam={isOnGlobalTeam}
      selectedTeamData={selectedTeamData}
    />
  );
};

const renderAllTeamsTable = (
  allTeamsScheduledQueriesList: IGlobalScheduledQuery[],
  allTeamsScheduledQueriesError: { name: string; reason: string }[],
  isOnGlobalTeam: boolean,
  selectedTeamData: ITeam | undefined
): JSX.Element => {
  if (Object.keys(allTeamsScheduledQueriesError).length > 0) {
    return <TableDataError />;
  }

  return (
    <div className={`${baseClass}__all-teams-table`}>
      <ScheduleListWrapper
        inheritedQueries
        allScheduledQueriesList={allTeamsScheduledQueriesList}
        isOnGlobalTeam={isOnGlobalTeam}
        selectedTeamData={selectedTeamData}
      />
    </div>
  );
};

interface ITeamSchedulesPageProps {
  params: {
    team_id: string;
  };
  location: any; // no type in react-router v3
}

// TODO: move team scheduled queries and global scheduled queries into services entities, remove redux
interface IRootState {
  entities: {
    global_scheduled_queries: {
      isLoading: boolean;
      data: IGlobalScheduledQuery[];
      errors: { name: string; reason: string }[];
    };
    team_scheduled_queries: {
      isLoading: boolean;
      data: ITeamScheduledQuery[];
      errors: { name: string; reason: string }[];
    };
  };
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
}: ITeamSchedulesPageProps): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS, MANAGE_SCHEDULE, MANAGE_TEAM_SCHEDULE } = paths;
  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  const {
    currentUser,
    isOnGlobalTeam,
    isPremiumTier,
    isFreeTier,
    currentTeam,
    setCurrentTeam,
  } = useContext(AppContext);

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

  const { data: teams, isLoading: isLoadingTeams } = useQuery(
    ["teams"],
    () => teamsAPI.loadAll({}),
    {
      enabled: !!isPremiumTier,
      select: (data) => {
        return currentUser?.teams
          ? filterAndSortTeamOptions(data.teams, currentUser.teams)
          : data.teams;
      },
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }
  );

  const { data: fleetQueries } = useQuery(
    ["fleetQueries"],
    () => fleetQueriesAPI.loadAll(),
    {
      select: (data) => data.queries,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }
  );

  let selectedTeamId: number;

  if (currentTeam) {
    selectedTeamId = currentTeam.id;
  } else {
    selectedTeamId = team_id ? parseInt(team_id, 10) : 0;
  }

  const handleTeamSelect = (teamId: number) => {
    if (teamId) {
      dispatch(push(MANAGE_TEAM_SCHEDULE(teamId)));
    } else {
      dispatch(push(MANAGE_SCHEDULE));
    }
    const selectedTeam = find(teams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  if (!isOnGlobalTeam && !selectedTeamId && teams) {
    handleTeamSelect(teams[0].id);
  }

  // TODO: move team scheduled queries and global scheduled queries into services entities, remove redux
  useEffect(() => {
    dispatch(
      selectedTeamId
        ? teamScheduledQueryActions.loadAll(selectedTeamId)
        : globalScheduledQueryActions.loadAll()
    );
  }, [dispatch, selectedTeamId]);

  const allScheduledQueries = useSelector((state: IRootState) => {
    if (selectedTeamId) {
      return state.entities.team_scheduled_queries;
    }
    return state.entities.global_scheduled_queries;
  });

  const allScheduledQueriesList = Object.values(allScheduledQueries.data);
  const allScheduledQueriesError = allScheduledQueries.errors;

  const allTeamsScheduledQueries = useSelector((state: IRootState) => {
    return state.entities.global_scheduled_queries;
  });

  const allTeamsScheduledQueriesList = Object.values(
    allTeamsScheduledQueries.data
  );
  const allTeamsScheduledQueriesError = allTeamsScheduledQueries.errors;

  const inheritedQueryOrQueries =
    allTeamsScheduledQueriesList.length === 1 ? "query" : "queries";

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
      return dispatch(
        selectedTeamId
          ? teamScheduledQueryActions.destroy(selectedTeamId, id)
          : globalScheduledQueryActions.destroy({ id })
      );
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
        dispatch(
          selectedTeamId
            ? teamScheduledQueryActions.loadAll(selectedTeamId)
            : globalScheduledQueryActions.loadAll()
        );
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
  ]);

  const onAddScheduledQuerySubmit = useCallback(
    (
      formData: IFormData,
      editQuery: IGlobalScheduledQuery | ITeamScheduledQuery | undefined
    ) => {
      if (editQuery) {
        const updatedAttributes = deepDifference(formData, editQuery);
        dispatch(
          selectedTeamId
            ? teamScheduledQueryActions.update(editQuery, updatedAttributes)
            : globalScheduledQueryActions.update(editQuery, updatedAttributes)
        )
          .then(() => {
            dispatch(
              renderFlash(
                "success",
                `Successfully updated ${formData.name} in the schedule.`
              )
            );
            dispatch(
              selectedTeamId
                ? teamScheduledQueryActions.loadAll(selectedTeamId)
                : globalScheduledQueryActions.loadAll()
            );
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
        dispatch(
          selectedTeamId
            ? teamScheduledQueryActions.create({ ...formData })
            : globalScheduledQueryActions.create({ ...formData })
        )
          .then(() => {
            dispatch(
              renderFlash(
                "success",
                `Successfully added ${formData.name} to the schedule.`
              )
            );
            dispatch(
              selectedTeamId
                ? teamScheduledQueryActions.loadAll(selectedTeamId)
                : globalScheduledQueryActions.loadAll()
            );
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
          {allScheduledQueriesList.length !== 0 &&
            allScheduledQueriesError.length !== 0 && (
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
              {!selectedTeamId ? (
                <p>
                  Schedule queries to run at regular intervals across{" "}
                  <strong>all of your hosts.</strong>
                </p>
              ) : (
                <p>
                  Schedule queries for{" "}
                  <strong>all hosts assigned to this team.</strong>
                </p>
              )}
            </div>
          )}
        </div>
        <div>
          {!isLoadingTeams &&
            renderTable(
              onRemoveScheduledQueryClick,
              onEditScheduledQueryClick,
              allScheduledQueriesList,
              allScheduledQueriesError,
              toggleScheduleEditorModal,
              isOnGlobalTeam || false,
              selectedTeamData
            )}
        </div>
        {/* must use ternary for NaN */}
        {selectedTeamId && allTeamsScheduledQueriesList.length > 0 ? (
          <>
            <span>
              <Button
                variant="unstyled"
                className={`${showInheritedQueries ? "upcarat" : "rightcarat"} 
                     ${baseClass}__inherited-queries-button`}
                onClick={toggleInheritedQueries}
              >
                {showInheritedQueries
                  ? `Hide ${allTeamsScheduledQueriesList.length} inherited ${inheritedQueryOrQueries}`
                  : `Show ${allTeamsScheduledQueriesList.length} inherited ${inheritedQueryOrQueries}`}
              </Button>
            </span>
            <div className={`${baseClass}__details`}>
              <IconToolTip
                isHtml
                text={
                  "\
              <center><p>Queries from the “All teams”<br/>schedule run on this team’s hosts.</p></center>\
            "
                }
              />
            </div>
          </>
        ) : null}
        {showInheritedQueries &&
          renderAllTeamsTable(
            allTeamsScheduledQueriesList,
            allTeamsScheduledQueriesError,
            isOnGlobalTeam || false,
            selectedTeamData
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
