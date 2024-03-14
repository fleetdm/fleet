import React, { useState, useCallback, useContext, useMemo } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { ITeam } from "interfaces/team";
import { IApiError } from "interfaces/errors";
import usersAPI, { IGetMeResponse } from "services/entities/users";
import teamsAPI, {
  ILoadTeamsResponse,
  ITeamFormData,
} from "services/entities/teams";

import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import SandboxGate from "components/Sandbox/SandboxGate";
import SandboxMessage from "components/Sandbox/SandboxMessage";

import CreateTeamModal from "./components/CreateTeamModal";
import DeleteTeamModal from "./components/DeleteTeamModal";
import RenameTeamModal from "./components/RenameTeamModal";
import EmptyTeamsTable from "./components/EmptyTeamsTable";

import { generateTableHeaders, generateDataSet } from "./TeamTableConfig";

const baseClass = "team-management";
const noTeamsClass = "no-teams";

const TeamManagementPage = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    currentTeam,
    setCurrentTeam,
    setCurrentUser,
    setAvailableTeams,
  } = useContext(AppContext);
  const [isUpdatingTeams, setIsUpdatingTeams] = useState(false);
  const [showCreateTeamModal, setShowCreateTeamModal] = useState(false);
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showRenameTeamModal, setShowRenameTeamModal] = useState(false);
  const [teamEditing, setTeamEditing] = useState<ITeam>();
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const handlePageError = useErrorHandler();

  const { refetch: refetchMe } = useQuery(["me"], () => usersAPI.me(), {
    enabled: false,
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
    },
  });

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

  // TODO: Cleanup useCallbacks, add missing dependencies, use state setter functions, e.g.,
  // `setShowCreateTeamModal((prevState) => !prevState)`, instead of including state
  // variables as dependencies for toggles, etc.

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

  const toggleRenameTeamModal = useCallback(
    (team?: ITeam) => {
      setShowRenameTeamModal(!showRenameTeamModal);
      setBackendValidators({});
      team ? setTeamEditing(team) : setTeamEditing(undefined);
    },
    [
      showRenameTeamModal,
      setShowRenameTeamModal,
      setTeamEditing,
      setBackendValidators,
    ]
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
          refetchMe();
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
    [toggleCreateTeamModal, refetchMe, refetchTeams, renderFlash]
  );

  const onDeleteSubmit = useCallback(() => {
    if (teamEditing) {
      setIsUpdatingTeams(true);
      teamsAPI
        .destroy(teamEditing.id)
        .then(() => {
          renderFlash("success", `Successfully deleted ${teamEditing.name}.`);
          if (currentTeam?.id === teamEditing.id) {
            setCurrentTeam(undefined);
          }
        })
        .catch(() => {
          renderFlash(
            "error",
            `Could not delete ${teamEditing.name}. Please try again.`
          );
        })
        .finally(() => {
          setIsUpdatingTeams(false);
          refetchMe();
          refetchTeams();
          toggleDeleteTeamModal();
        });
    }
  }, [
    currentTeam,
    teamEditing,
    refetchMe,
    refetchTeams,
    renderFlash,
    setCurrentTeam,
    toggleDeleteTeamModal,
  ]);

  const onRenameSubmit = useCallback(
    (formData: ITeamFormData) => {
      if (formData.name === teamEditing?.name) {
        toggleRenameTeamModal();
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
            toggleRenameTeamModal();
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
                `Could not rename ${teamEditing.name}. Please try again.`
              );
            }
          })
          .finally(() => {
            setIsUpdatingTeams(false);
          });
      }
    },
    [teamEditing, toggleRenameTeamModal, refetchTeams, renderFlash]
  );

  const onActionSelection = useCallback(
    (action: string, team: ITeam): void => {
      switch (action) {
        case "rename":
          toggleRenameTeamModal(team);
          break;
        case "delete":
          toggleDeleteTeamModal(team);
          break;
        default:
      }
    },
    [toggleRenameTeamModal, toggleDeleteTeamModal]
  );

  const tableHeaders = useMemo(() => generateTableHeaders(onActionSelection), [
    onActionSelection,
  ]);
  const tableData = useMemo(() => (teams ? generateDataSet(teams) : []), [
    teams,
  ]);

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Create, customize, and remove teams from Fleet.
      </p>
      <SandboxGate
        fallbackComponent={() => (
          <SandboxMessage
            variant="sales"
            message="Teams is only available in Fleet premium."
            utmSource="fleet-ui-teams-page"
            className={`${baseClass}__sandbox-message`}
          />
        )}
      >
        {loadingTeamsError ? (
          <TableDataError />
        ) : (
          <TableContainer
            columnConfigs={tableHeaders}
            data={tableData}
            isLoading={isFetchingTeams}
            defaultSortHeader={"name"}
            defaultSortDirection={"asc"}
            actionButton={{
              name: "create team",
              buttonText: "Create team",
              variant: "brand",
              onActionButtonClick: toggleCreateTeamModal,
              hideButton: teams && teams.length === 0,
            }}
            resultsTitle={"teams"}
            emptyComponent={() => (
              <EmptyTeamsTable
                className={noTeamsClass}
                onActionButtonClick={toggleCreateTeamModal}
              />
            )}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            isClientSidePagination
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
        {showRenameTeamModal && (
          <RenameTeamModal
            onCancel={toggleRenameTeamModal}
            onSubmit={onRenameSubmit}
            defaultName={teamEditing?.name || ""}
            backendValidators={backendValidators}
            isUpdatingTeams={isUpdatingTeams}
          />
        )}
      </SandboxGate>
    </div>
  );
};

export default TeamManagementPage;
