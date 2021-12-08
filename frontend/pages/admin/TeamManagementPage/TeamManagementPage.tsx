import React, { useState, useCallback } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";

import { ITeam } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

import teamsAPI from "services/entities/teams";

import TableContainer from "components/TableContainer";
// @ts-ignore
import TableDataError from "components/TableDataError";
import CreateTeamModal from "./components/CreateTeamModal";
import DeleteTeamModal from "./components/DeleteTeamModal";
import EditTeamModal from "./components/EditTeamModal";
import { ICreateTeamFormData } from "./components/CreateTeamModal/CreateTeamModal";
import { IEditTeamFormData } from "./components/EditTeamModal/EditTeamModal";
import { generateTableHeaders, generateDataSet } from "./TeamTableConfig";

interface ITeamsResponse {
  teams: ITeam[];
}

const baseClass = "team-management";
const noTeamsClass = "no-teams";

const TeamManagementPage = (): JSX.Element => {
  const dispatch = useDispatch();
  const [showCreateTeamModal, setShowCreateTeamModal] = useState(false);
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);
  const [teamEditing, setTeamEditing] = useState<ITeam>();
  const [searchString, setSearchString] = useState<string>("");

  const {
    data: teams,
    isLoading: isLoadingTeams,
    error: loadingTeamsError,
    refetch: refetchTeams,
  } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ITeamsResponse) => data.teams,
    }
  );

  const toggleCreateTeamModal = useCallback(() => {
    setShowCreateTeamModal(!showCreateTeamModal);
  }, [showCreateTeamModal, setShowCreateTeamModal]);

  const toggleDeleteTeamModal = useCallback(
    (team?: ITeam) => {
      setShowDeleteTeamModal(!showDeleteTeamModal);
      team ? setTeamEditing(team) : setTeamEditing(undefined);
    },
    [showDeleteTeamModal, setShowDeleteTeamModal, setTeamEditing]
  );

  const toggleEditTeamModal = useCallback(
    (team?: ITeam) => {
      setShowEditTeamModal(!showEditTeamModal);
      team ? setTeamEditing(team) : setTeamEditing(undefined);
    },
    [showEditTeamModal, setShowEditTeamModal, setTeamEditing]
  );

  // NOTE: called once on the initial render of this component.
  const onQueryChange = useCallback(
    (queryData) => {
      setSearchString(queryData.searchQuery);
      const { pageIndex, pageSize, searchQuery } = queryData;
      teamsAPI.loadAll({
        page: pageIndex,
        perPage: pageSize,
        globalFilter: searchQuery,
      });
    },
    [dispatch, setSearchString]
  );

  const onCreateSubmit = useCallback(
    (formData: ICreateTeamFormData) => {
      teamsAPI
        .create(formData)
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully created ${formData.name}.`)
          );
          toggleCreateTeamModal();
        })
        .catch((createError: any) => {
          console.error(createError);
          if (createError.errors[0].reason.includes("Duplicate")) {
            dispatch(
              renderFlash("error", "A team with this name already exists.")
            );
          } else {
            dispatch(
              renderFlash("error", "Could not create team. Please try again.")
            );
          }
        })
        .finally(() => {
          refetchTeams();
        });
    },
    [dispatch, toggleCreateTeamModal]
  );

  const onDeleteSubmit = useCallback(() => {
    if (teamEditing) {
      teamsAPI
        .destroy(teamEditing.id)
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully deleted ${teamEditing.name}.`)
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not delete ${teamEditing.name}. Please try again.`
            )
          );
        })
        .finally(() => {
          refetchTeams();
          toggleDeleteTeamModal();
        });
    }
  }, [dispatch, teamEditing, toggleDeleteTeamModal]);

  const onEditSubmit = useCallback(
    (formData: IEditTeamFormData) => {
      if (formData.name === teamEditing?.name) {
        toggleEditTeamModal();
      } else if (teamEditing) {
        teamsAPI
          .update(teamEditing.id, formData)
          .then(() => {
            dispatch(
              renderFlash("success", `Successfully edited ${formData.name}.`)
            );
          })
          .catch((updateError) => {
            console.error(updateError);
            if (updateError.errors[0].reason.includes("Duplicate")) {
              dispatch(
                renderFlash("error", "A team with this name already exists.")
              );
            } else {
              dispatch(
                renderFlash(
                  "error",
                  `Could not edit ${teamEditing.name}. Please try again.`
                )
              );
            }
          })
          .finally(() => {
            refetchTeams();
            toggleEditTeamModal();
          });
      }
    },
    [dispatch, teamEditing, toggleEditTeamModal]
  );

  const onActionSelection = (action: string, team: ITeam): void => {
    switch (action) {
      case "edit":
        toggleEditTeamModal(team);
        break;
      case "delete":
        toggleDeleteTeamModal(team);
        break;
      default:
    }
  };

  const NoTeamsComponent = () => {
    return (
      <div className={`${noTeamsClass}`}>
        <div className={`${noTeamsClass}__inner`}>
          <div className={`${noTeamsClass}__inner-text`}>
            <h1>Set up team permissions</h1>
            <p>
              Keep your organization organized and efficient by ensuring every
              user has the correct access to the right hosts.
            </p>
            <p>
              Want to learn more?&nbsp;
              <a
                href="https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/10-Teams.md"
                target="_blank"
                rel="noopener noreferrer"
              >
                Read about teams&nbsp;
                <FleetIcon name="external-link" />
              </a>
            </p>
            <Button
              variant="brand"
              className={`${noTeamsClass}__create-button`}
              onClick={toggleCreateTeamModal}
            >
              Create team
            </Button>
          </div>
        </div>
      </div>
    );
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const tableData = teams ? generateDataSet(teams) : [];

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Create, customize, and remove teams from Fleet.
      </p>
      {loadingTeamsError ? (
        <TableDataError />
      ) : (
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={isLoadingTeams}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search"}
          actionButtonText={"Create team"}
          actionButtonVariant={"brand"}
          hideActionButton={teams && teams.length === 0 && searchString === ""}
          onActionButtonClick={toggleCreateTeamModal}
          onQueryChange={onQueryChange}
          resultsTitle={"teams"}
          emptyComponent={NoTeamsComponent}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={teams && teams.length > 0 && searchString !== ""}
          isClientSideSearch
        />
      )}
      {showCreateTeamModal ? (
        <CreateTeamModal
          onCancel={toggleCreateTeamModal}
          onSubmit={onCreateSubmit}
        />
      ) : null}
      {showDeleteTeamModal ? (
        <DeleteTeamModal
          onCancel={toggleDeleteTeamModal}
          onSubmit={onDeleteSubmit}
          name={teamEditing?.name || ""}
        />
      ) : null}
      {showEditTeamModal ? (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          onSubmit={onEditSubmit}
          defaultName={teamEditing?.name || ""}
        />
      ) : null}
    </div>
  );
};

export default TeamManagementPage;
