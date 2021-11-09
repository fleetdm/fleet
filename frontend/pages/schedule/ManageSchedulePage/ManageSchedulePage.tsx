/* Conditionally renders global schedule and team schedules */

import React, { useState, useCallback, useEffect, useContext } from "react";
import { useQuery } from "react-query";
import { useDispatch, useSelector } from "react-redux";
import { AppContext } from "context/app";

import { push } from "react-router-redux";
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

import paths from "router/paths";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
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

interface ITeamOptions {
  disabled: boolean;
  label: string;
  value: string | number;
}

const ManageSchedulePage = ({
  params: { team_id },
}: ITeamSchedulesPageProps): JSX.Element => {
  let teamId = parseInt(team_id, 10);
  const dispatch = useDispatch();
  const { MANAGE_PACKS, MANAGE_SCHEDULE, MANAGE_TEAM_SCHEDULE } = paths;
  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  const {
    currentUser,
    isOnGlobalTeam,
    isPremiumTier,
    isTeamMaintainerOrTeamAdmin,
  } = useContext(AppContext);

  const onChangeSelectedTeam = (selectedTeamId: number) => {
    if (isNaN(selectedTeamId)) {
      dispatch(push(MANAGE_SCHEDULE));
    } else {
      dispatch(push(MANAGE_TEAM_SCHEDULE(selectedTeamId)));
    }
  };

  const loadFirstMaintainerOrAdminTeam = () => {
    if (currentUser) {
      const adminOrMaintainerTeam = currentUser.teams.find((team) => {
        return team.role === "admin" || team.role === "maintainer"
          ? team.id
          : null;
      });
      if (adminOrMaintainerTeam) {
        teamId = adminOrMaintainerTeam.id;
        onChangeSelectedTeam(teamId);
      }
    }
  };

  if (!isOnGlobalTeam && !isTeamMaintainerOrTeamAdmin && !teamId) {
    loadFirstMaintainerOrAdminTeam();
  }

  if (!isOnGlobalTeam && !isTeamMaintainerOrTeamAdmin && teamId) {
    if (currentUser) {
      const canLoadTeam = currentUser.teams.find((team) => {
        return (
          (team.role === "admin" || team.role === "maintainer") &&
          team.id === teamId
        );
      });
      if (!canLoadTeam) {
        loadFirstMaintainerOrAdminTeam();
      }
    }
  }

  const { data: teams } = useQuery(["teams"], () => teamsAPI.loadAll({}), {
    enabled: !!isPremiumTier,
    select: (data) => data.teams,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  const { data: fleetQueries } = useQuery(
    ["fleetQueries"],
    () => fleetQueriesAPI.loadAll(),
    {
      select: (data) => data.queries,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }
  );

  // TODO: move team scheduled queries and global scheduled queries into services entities, remove redux
  useEffect(() => {
    dispatch(
      teamId
        ? teamScheduledQueryActions.loadAll(teamId)
        : globalScheduledQueryActions.loadAll()
    );
  }, [dispatch, teamId]);

  const allScheduledQueries = useSelector((state: IRootState) => {
    if (teamId) {
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

  const selectedTeam = isNaN(teamId) ? "global" : teamId;

  const selectedTeamData = allTeamsList.find(
    (team) => selectedTeam === team.id
  );

  const [showInheritedQueries, setShowInheritedQueries] = useState<boolean>(
    false
  );
  const [showScheduleEditorModal, setShowScheduleEditorModal] = useState(false);
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

  const toggleScheduleEditorModal = useCallback(() => {
    setSelectedScheduledQuery(undefined); // create modal renders
    setShowScheduleEditorModal(!showScheduleEditorModal);
  }, [showScheduleEditorModal, setShowScheduleEditorModal]);

  const toggleRemoveScheduledQueryModal = useCallback(() => {
    setShowRemoveScheduledQueryModal(!showRemoveScheduledQueryModal);
  }, [showRemoveScheduledQueryModal, setShowRemoveScheduledQueryModal]);

  const generateTeamOptionsDropdownItems = (): ITeamOptions[] => {
    const teamOptions: ITeamOptions[] = [];

    if (isTeamMaintainerOrTeamAdmin && currentUser) {
      currentUser.teams.forEach((team) => {
        if (team.role === "admin" || team.role === "maintainer") {
          teamOptions.push({
            disabled: false,
            label: team.name,
            value: team.id,
          });
        }
      });
    } else if (isOnGlobalTeam && teams) {
      teamOptions.push({
        disabled: false,
        label: "All teams",
        value: "global",
      });

      teams.forEach((team: ITeam) => {
        teamOptions.push({
          disabled: false,
          label: team.name,
          value: team.id,
        });
      });
    }

    return teamOptions;
  };

  const renderTitleOrDropdown = (): JSX.Element => {
    const dropDownOptions = generateTeamOptionsDropdownItems();
    return dropDownOptions.length === 1 ? (
      <h1>{dropDownOptions[0].label}</h1>
    ) : (
      <Dropdown
        value={selectedTeam}
        className={`${baseClass}__team-dropdown`}
        options={dropDownOptions}
        searchable={false}
        onChange={(newSelectedValue: number) =>
          onChangeSelectedTeam(newSelectedValue)
        }
      />
    );
  };

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
        teamId
          ? teamScheduledQueryActions.destroy(teamId, id)
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
          teamId
            ? teamScheduledQueryActions.loadAll(teamId)
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
  }, [dispatch, teamId, selectedQueryIds, toggleRemoveScheduledQueryModal]);

  const onAddScheduledQuerySubmit = useCallback(
    (
      formData: IFormData,
      editQuery: IGlobalScheduledQuery | ITeamScheduledQuery | undefined
    ) => {
      if (editQuery) {
        const updatedAttributes = deepDifference(formData, editQuery);
        dispatch(
          teamId
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
              teamId
                ? teamScheduledQueryActions.loadAll(teamId)
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
          teamId
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
              teamId
                ? teamScheduledQueryActions.loadAll(teamId)
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
    [dispatch, teamId, toggleScheduleEditorModal]
  );

  if (selectedTeam === "global" && isTeamMaintainerOrTeamAdmin) {
    const teamMaintainerTeams = generateTeamOptionsDropdownItems();
    if (teamMaintainerTeams.length) {
      dispatch(
        push(MANAGE_TEAM_SCHEDULE(Number(teamMaintainerTeams[0].value)))
      );
    }
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            {!isPremiumTier ? (
              <div className={`${baseClass}__text`}>
                <h1 className={`${baseClass}__title`}>
                  <span>Schedule</span>
                </h1>
                <div className={`${baseClass}__description`}>
                  <p>
                    Schedule recurring queries for your hosts. Fleet’s query
                    schedule lets you add queries which are executed at regular
                    intervals.
                  </p>
                </div>
              </div>
            ) : (
              <div>
                {renderTitleOrDropdown()}
                <div className={`${baseClass}__description`}>
                  {isNaN(teamId) ? (
                    <p>
                      Schedule queries to run at regular intervals across{" "}
                      <b>all of your hosts</b>.
                    </p>
                  ) : (
                    <p>
                      Schedule additional queries for all hosts assigned to this
                      team.
                    </p>
                  )}
                </div>
              </div>
            )}
          </div>
          {/* Hide CTA Buttons if no schedule or schedule error */}
          {allScheduledQueriesList.length !== 0 &&
            allScheduledQueriesError.length !== 0 && (
              <div className={`${baseClass}__action-button-container`}>
                {!isTeamMaintainerOrTeamAdmin && (
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
        <div>
          {renderTable(
            onRemoveScheduledQueryClick,
            onEditScheduledQueryClick,
            allScheduledQueriesList,
            allScheduledQueriesError,
            toggleScheduleEditorModal,
<<<<<<< HEAD
            isOnGlobalTeam || false,
            selectedTeamData
=======
            teamId,
            isTeamMaintainerOrTeamAdmin || false,
            isOnGlobalTeam || false
>>>>>>> 5d87ba70 (Refactor schedule page to new patterns except global and team scheduled queries API)
          )}
        </div>
        {/* must use ternary for NaN */}
        {teamId && allTeamsScheduledQueriesList.length > 0 ? (
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
<<<<<<< HEAD
            isOnGlobalTeam || false,
            selectedTeamData
=======
            teamId,
            isTeamMaintainerOrTeamAdmin || false,
            isOnGlobalTeam || false
>>>>>>> 5d87ba70 (Refactor schedule page to new patterns except global and team scheduled queries API)
          )}
        {showScheduleEditorModal && (
          <ScheduleEditorModal
            onCancel={toggleScheduleEditorModal}
            onScheduleSubmit={onAddScheduledQuerySubmit}
            allQueries={fleetQueries}
            editQuery={selectedScheduledQuery}
            teamId={teamId}
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
