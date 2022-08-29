import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";

import { NotificationContext } from "context/notification";
import { ITeam } from "interfaces/team";
import { IApiError } from "interfaces/errors";
import teamsAPI, {
  ILoadTeamsResponse,
  ITeamFormData,
} from "services/entities/teams";

import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import CreateTeamModal from "./components/CreateTeamModal";
import DeleteTeamModal from "./components/DeleteTeamModal";
import EditTeamModal from "./components/EditTeamModal";
import { generateTableHeaders, generateDataSet } from "./TeamTableConfig";

const baseClass = "team-management";
const noTeamsClass = "no-teams";

const TeamManagementPage = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdatingTeams, setIsUpdatingTeams] = useState<boolean>(false);
  const [showCreateTeamModal, setShowCreateTeamModal] = useState(false);
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);
  const [teamEditing, setTeamEditing] = useState<ITeam>();
  const [searchString, setSearchString] = useState<string>("");
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const handlePageError = useErrorHandler();

  const {
    data: teams,
    isFetching: isFetchingTeams,
    error: loadingTeamsError,
    refetch: refetchTeams,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ILoadTeamsResponse) => data.teams,
      onError: (error) => handlePageError(error),
    }
  );

  const toggleCreateTeamModal = useCallback(() => {
    setShowCreateTeamModal(!showCreateTeamModal);
    setBackendValidators({});
  }, [showCreateTeamModal, setShowCreateTeamModal, setBackendValidators]);

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
      setBackendValidators({});
      team ? setTeamEditing(team) : setTeamEditing(undefined);
    },
    [
      showEditTeamModal,
      setShowEditTeamModal,
      setTeamEditing,
      setBackendValidators,
    ]
  );

  const onQueryChange = useCallback(
    (queryData) => {
      if (teams) {
        setSearchString(queryData.searchQuery);
        const { pageIndex, pageSize, searchQuery } = queryData;
        teamsAPI.loadAll({
          page: pageIndex,
          perPage: pageSize,
          globalFilter: searchQuery,
        });
      }
    },
    [setSearchString]
  );

  const onCreateSubmit = useCallback(
    (formData: ITeamFormData) => {
      setIsUpdatingTeams(true);
      teamsAPI
        .create(formData)
        .then(() => {
          renderFlash("success", `Successfully created ${formData.name}.`);
          setBackendValidators({});
          toggleCreateTeamModal();
          refetchTeams();
        })
        .catch((createError: { data: IApiError }) => {
          if (createError.data.errors[0].reason.includes("Duplicate")) {
            setBackendValidators({
              name: "A team with this name already exists",
            });
          } else {
            renderFlash("error", "Could not create team. Please try again.");
            toggleCreateTeamModal();
          }
        })
        .finally(() => {
          setIsUpdatingTeams(false);
        });
    },
    [toggleCreateTeamModal]
  );

  const onDeleteSubmit = useCallback(() => {
    if (teamEditing) {
      setIsUpdatingTeams(true);
      teamsAPI
        .destroy(teamEditing.id)
        .then(() => {
          renderFlash("success", `Successfully deleted ${teamEditing.name}.`);
        })
        .catch(() => {
          renderFlash(
            "error",
            `Could not delete ${teamEditing.name}. Please try again.`
          );
        })
        .finally(() => {
          setIsUpdatingTeams(false);
          refetchTeams();
          toggleDeleteTeamModal();
        });
    }
  }, [teamEditing, toggleDeleteTeamModal]);

  const onEditSubmit = useCallback(
    (formData: ITeamFormData) => {
      if (formData.name === teamEditing?.name) {
        toggleEditTeamModal();
      } else if (teamEditing) {
        setIsUpdatingTeams(true);
        teamsAPI
          .update(formData, teamEditing.id)
          .then(() => {
            renderFlash(
              "success",
              `Successfully updated team name to ${formData.name}.`
            );
            setBackendValidators({});
            toggleEditTeamModal();
            refetchTeams();
          })
          .catch((updateError: { data: IApiError }) => {
            console.error(updateError);
            if (updateError.data.errors[0].reason.includes("Duplicate")) {
              setBackendValidators({
                name: "A team with this name already exists",
              });
            } else {
              renderFlash(
                "error",
                `Could not edit ${teamEditing.name}. Please try again.`
              );
            }
          })
          .finally(() => {
            setIsUpdatingTeams(false);
          });
      }
    },
    [teamEditing, toggleEditTeamModal]
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
                href="https://fleetdm.com/docs/using-fleet/teams"
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
          isLoading={isFetchingTeams}
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
          disablePagination
        />
      )}
      {showCreateTeamModal && (
        <CreateTeamModal
          onCancel={toggleCreateTeamModal}
          onSubmit={onCreateSubmit}
          backendValidators={backendValidators}
          isUpdatingTeams={isUpdatingTeams}
        />
      )}
      {showDeleteTeamModal && (
        <DeleteTeamModal
          onCancel={toggleDeleteTeamModal}
          onSubmit={onDeleteSubmit}
          name={teamEditing?.name || ""}
          isUpdatingTeams={isUpdatingTeams}
        />
      )}
      {showEditTeamModal && (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          onSubmit={onEditSubmit}
          defaultName={teamEditing?.name || ""}
          backendValidators={backendValidators}
          isUpdatingTeams={isUpdatingTeams}
        />
      )}
    </div>
  );
};

export default TeamManagementPage;
