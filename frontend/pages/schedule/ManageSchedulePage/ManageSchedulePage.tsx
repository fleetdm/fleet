/* Conditionally renders global schedule and team schedules */

import React, { useState, useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";

import { push } from "react-router-redux";
// @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IConfig } from "interfaces/config";
import { IQuery } from "interfaces/query";
import { ITeam } from "interfaces/team";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import { ITeamScheduledQuery } from "interfaces/team_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";
// @ts-ignore
import teamScheduledQueryActions from "redux/nodes/entities/team_scheduled_queries/actions";
// @ts-ignore
import queryActions from "redux/nodes/entities/queries/actions";
import teamActions from "redux/nodes/entities/teams/actions";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import paths from "router/paths";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import ScheduleError from "./components/ScheduleError";
import ScheduleListWrapper from "./components/ScheduleListWrapper";
import ScheduleEditorModal from "./components/ScheduleEditorModal";
import RemoveScheduledQueryModal from "./components/RemoveScheduledQueryModal";

const baseClass = "manage-schedule-page";

const renderTable = (
  onRemoveScheduledQueryClick: React.MouseEventHandler<HTMLButtonElement>,
  onEditScheduledQueryClick: React.MouseEventHandler<HTMLButtonElement>,
  allScheduledQueriesList: IGlobalScheduledQuery[] | ITeamScheduledQuery[],
  allScheduledQueriesError: any,
  toggleScheduleEditorModal: () => void,
  teamId: number
): JSX.Element => {
  if (Object.keys(allScheduledQueriesError).length !== 0) {
    return <ScheduleError />;
  }

  return (
    <ScheduleListWrapper
      onRemoveScheduledQueryClick={onRemoveScheduledQueryClick}
      onEditScheduledQueryClick={onEditScheduledQueryClick}
      allScheduledQueriesList={allScheduledQueriesList}
      toggleScheduleEditorModal={toggleScheduleEditorModal}
      teamId={teamId}
    />
  );
};

interface ITeamSchedulesPageProps {
  params: {
    team_id: string;
  };
}
interface IRootState {
  app: {
    config: IConfig;
  };
  entities: {
    global_scheduled_queries: {
      isLoading: boolean;
      data: IGlobalScheduledQuery[];
      errors: any;
    };
    team_scheduled_queries: {
      isLoading: boolean;
      data: ITeamScheduledQuery[];
      errors: any;
    };
    queries: {
      isLoading: boolean;
      data: IQuery[];
    };
    teams: {
      isLoading: boolean;
      data: ITeam[];
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

const ManageSchedulePage = (props: ITeamSchedulesPageProps): JSX.Element => {
  const {
    params: { team_id },
  } = props;
  const teamId = parseInt(team_id, 10);
  const dispatch = useDispatch();
  const { MANAGE_PACKS } = paths;
  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  useEffect(() => {
    dispatch(queryActions.loadAll());
    dispatch(teamActions.loadAll());
    dispatch(
      teamId
        ? teamScheduledQueryActions.loadAll(teamId)
        : globalScheduledQueryActions.loadAll()
    );
  }, [dispatch, teamId]);

  const isBasicTier = useSelector((state: IRootState) => {
    return state.app.config.tier === "basic";
  });

  const allQueries = useSelector((state: IRootState) => state.entities.queries);
  const allQueriesList = Object.values(allQueries.data);

  const allScheduledQueries = useSelector((state: IRootState) => {
    if (teamId) {
      return state.entities.team_scheduled_queries;
    }
    return state.entities.global_scheduled_queries;
  });

  const allScheduledQueriesList = Object.values(allScheduledQueries.data);
  const allScheduledQueriesError = allScheduledQueries.errors;

  const allTeams = useSelector((state: IRootState) => state.entities.teams);
  const allTeamsList = Object.values(allTeams.data);

  const selectedTeam = isNaN(teamId) ? "global" : teamId;

  const generateTeamOptionsDropdownItems = (): ITeamOptions[] => {
    const teamOptions: ITeamOptions[] = [
      {
        disabled: false,
        label: "Global",
        value: "global",
      },
    ];

    allTeamsList.forEach((team) => {
      teamOptions.push({
        disabled: false,
        label: team.name,
        value: team.id,
      });
    });
    return teamOptions;
  };

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

  const toggleScheduleEditorModal = useCallback(() => {
    setSelectedScheduledQuery(undefined); // create modal renders
    setShowScheduleEditorModal(!showScheduleEditorModal);
  }, [showScheduleEditorModal, setShowScheduleEditorModal]);

  const toggleRemoveScheduledQueryModal = useCallback(() => {
    setShowRemoveScheduledQueryModal(!showRemoveScheduledQueryModal);
  }, [showRemoveScheduledQueryModal, setShowRemoveScheduledQueryModal]);

  const onRemoveScheduledQueryClick = (selectedTableQueryIds: any): void => {
    toggleRemoveScheduledQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onEditScheduledQueryClick = (selectedQuery: any): void => {
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

  const onChangeSelectedTeam = (selectedTeamId: number) => {
    if (isNaN(selectedTeamId)) {
      dispatch(push(`${paths.MANAGE_SCHEDULE}`));
    } else {
      dispatch(push(`${paths.MANAGE_TEAM_SCHEDULE(selectedTeamId)}`));
    }
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            {!isBasicTier ? (
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
                <Dropdown
                  value={selectedTeam}
                  className={`${baseClass}__team-dropdown`}
                  options={generateTeamOptionsDropdownItems()}
                  searchable={false}
                  onChange={(newSelectedValue: number) =>
                    onChangeSelectedTeam(newSelectedValue)
                  }
                />
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
                <Button
                  variant="inverse"
                  onClick={handleAdvanced}
                  className={`${baseClass}__advanced-button`}
                >
                  Advanced
                </Button>
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
            teamId
          )}
        </div>
        {showScheduleEditorModal && (
          <ScheduleEditorModal
            onCancel={toggleScheduleEditorModal}
            onScheduleSubmit={onAddScheduledQuerySubmit}
            allQueries={allQueriesList}
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
