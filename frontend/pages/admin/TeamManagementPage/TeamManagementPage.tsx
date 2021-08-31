import React, { useState, useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";

import { ITeam } from "interfaces/team";
import teamActions from "redux/nodes/entities/teams/actions";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import TableContainer from "components/TableContainer";

import TableDataError from "components/TableDataError";
import CreateTeamModal from "./components/CreateTeamModal";
import DeleteTeamModal from "./components/DeleteTeamModal";
import EditTeamModal from "./components/EditTeamModal";
import { ICreateTeamFormData } from "./components/CreateTeamModal/CreateTeamModal";
import { IEditTeamFormData } from "./components/EditTeamModal/EditTeamModal";
import { generateTableHeaders, generateDataSet } from "./TeamTableConfig";

const baseClass = "team-management";
const noTeamsClass = "no-teams";

// TODO: should probably live close to the store.js file and imported in.
interface IRootState {
  entities: {
    teams: {
      isLoading: boolean;
      data: { [id: number]: ITeam };
      errors: { name: string; reason: string }[];
    };
  };
}

const generateUpdateData = (
  currentTeamData: ITeam,
  formData: IEditTeamFormData
): IEditTeamFormData | null => {
  if (currentTeamData.name !== formData.name) {
    return {
      name: formData.name,
    };
  }
  return null;
};

const TeamManagementPage = (): JSX.Element => {
  const dispatch = useDispatch();
  const [showCreateTeamModal, setShowCreateTeamModal] = useState(false);
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);
  const [teamEditing, setTeamEditing] = useState<ITeam>();
  const [searchString, setSearchString] = useState<string>("");

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
      dispatch(
        teamActions.loadAll({
          page: pageIndex,
          perPage: pageSize,
          globalFilter: searchQuery,
        })
      );
    },
    [dispatch, setSearchString]
  );

  const onCreateSubmit = useCallback(
    (formData: ICreateTeamFormData) => {
      dispatch(teamActions.create(formData))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully created ${formData.name}.`)
          );
          dispatch(teamActions.loadAll({}));
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not create team. Please try again.")
          );
        });
      toggleCreateTeamModal();
    },
    [dispatch, toggleCreateTeamModal]
  );

  const onDeleteSubmit = useCallback(() => {
    dispatch(teamActions.destroy(teamEditing?.id))
      .then(() => {
        dispatch(
          renderFlash("success", `Successfully deleted ${teamEditing?.name}.`)
        );
        dispatch(teamActions.loadAll({}));
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Could not delete ${teamEditing?.name}. Please try again.`
          )
        );
      });
    toggleDeleteTeamModal();
  }, [dispatch, teamEditing, toggleDeleteTeamModal]);

  const onEditSubmit = useCallback(
    (formData: IEditTeamFormData) => {
      const updatedAttrs = generateUpdateData(teamEditing as ITeam, formData);
      // no updates, so no need for a request.
      if (updatedAttrs === null) {
        toggleEditTeamModal();
        return;
      }
      dispatch(teamActions.update(teamEditing?.id, updatedAttrs))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully edited ${formData.name}.`)
          );
          dispatch(teamActions.loadAll({}));
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not edit ${teamEditing?.name}. Please try again.`
            )
          );
        });
      toggleEditTeamModal();
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

  const NoTeamsComponent = useCallback(() => {
    return (
      <div className={`${noTeamsClass}`}>
        <div className={`${noTeamsClass}__inner`}>
          <div className={`${noTeamsClass}__inner-text`}>
            {searchString === "" ? (
              <>
                <h1>Set up team permissions</h1>
                <p>
                  Keep your organization organized and efficient by ensuring
                  every user has the correct access to the right hosts.
                </p>
                <p>
                  Want to learn more?&nbsp;
                  <a
                    href="https://github.com/fleetdm/fleet/tree/master/docs/1-Using-Fleet/role-based-access-control-and-teams.md"
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
                  Create Team
                </Button>
              </>
            ) : (
              <>
                <h2>No members match the current search criteria.</h2>
                <p>
                  Expecting to see members? Try again in a few seconds as the
                  system catches up.
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    );
  }, [searchString]);

  const tableHeaders = generateTableHeaders(onActionSelection);
  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.teams.isLoading
  );
  const teams = useSelector((state: IRootState) =>
    generateDataSet(state.entities.teams.data)
  );

  const teamsError = useSelector(
    (state: IRootState) => state.entities.teams.errors
  );

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Create, customize, and remove teams from Fleet.
      </p>
      {Object.keys(teamsError).length > 0 ? (
        <TableDataError />
      ) : (
        <TableContainer
          columns={tableHeaders}
          data={teams}
          isLoading={loadingTableData}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search"}
          actionButtonText={"Create team"}
          actionButtonVariant={"primary"}
          hideActionButton={teams.length === 0 && searchString === ""}
          onActionButtonClick={toggleCreateTeamModal}
          onQueryChange={onQueryChange}
          resultsTitle={"teams"}
          emptyComponent={NoTeamsComponent}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={teams.length > 0 && searchString !== ""}
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
