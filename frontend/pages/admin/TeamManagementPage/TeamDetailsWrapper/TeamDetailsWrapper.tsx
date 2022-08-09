import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, Link, RouteProps } from "react-router";
import { Tab, TabList, Tabs } from "react-tabs";
import { find, toNumber } from "lodash";
import classnames from "classnames";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import PATHS from "router/paths";
import { ITeam, ITeamSummary } from "interfaces/team";
import teamsAPI, {
  ILoadTeamsResponse,
  ITeamFormData,
} from "services/entities/teams";
import usersAPI, { IGetMeResponse } from "services/entities/users";
import enrollSecretsAPI from "services/entities/enroll_secret";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import permissions from "utilities/permissions";
import sortUtils from "utilities/sort";
import ActionButtons from "components/buttons/ActionButtons/ActionButtons";
import formatErrorResponse from "utilities/format_error_response";

import Spinner from "components/Spinner";
import TabsWrapper from "components/TabsWrapper";
import TeamsDropdown from "components/TeamsDropdown";
import MainContent from "components/MainContent";
import { getNextLocationPath } from "pages/admin/UserManagementPage/helpers/userManagementHelpers";
import DeleteTeamModal from "../components/DeleteTeamModal";
import EditTeamModal from "../components/EditTeamModal";
import DeleteSecretModal from "../../../../components/DeleteSecretModal";
import SecretEditorModal from "../../../../components/SecretEditorModal";
import AddHostsModal from "../../../../components/AddHostsModal";
import EnrollSecretModal from "../../../../components/EnrollSecretModal";

import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";
import EyeIcon from "../../../../../assets/images/icon-eye-16x16@2x.png";
import PencilIcon from "../../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../../assets/images/icon-trash-14x14@2x.png";

const baseClass = "team-details";

interface ITeamDetailsSubNavItem {
  name: string;
  getPathname: (id: number) => string;
}

const teamDetailsSubNav: ITeamDetailsSubNavItem[] = [
  {
    name: "Members",
    getPathname: PATHS.TEAM_DETAILS_MEMBERS,
  },
  {
    name: "Agent options",
    getPathname: PATHS.TEAM_DETAILS_OPTIONS,
  },
];

interface ITeamDetailsPageProps {
  children: JSX.Element;
  params: {
    team_id: string;
  };
  location: {
    pathname: string;
  };
  route: RouteProps;
  router: InjectedRouter;
}

const generateUpdateData = (
  currentTeam: ITeamSummary,
  formData: ITeamFormData
): ITeamFormData | null => {
  if (currentTeam.name !== formData.name) {
    return {
      name: formData.name,
    };
  }
  return null;
};

const getTabIndex = (path: string, teamId: number): number => {
  return teamDetailsSubNav.findIndex((navItem) => {
    return navItem.getPathname(teamId).includes(path);
  });
};

const TeamDetailsWrapper = ({
  route,
  router,
  children,
  location: { pathname },
  params: routeParams,
}: ITeamDetailsPageProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const handlePageError = useErrorHandler();
  const teamIdFromURL = parseInt(routeParams.team_id, 10) || 0;
  const {
    availableTeams,
    currentUser,
    isGlobalAdmin,
    currentTeam,
    isPremiumTier,
    setAvailableTeams,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);

  const routeTemplate = route && route.path ? route.path : "";

  const [selectedSecret, setSelectedSecret] = useState<IEnrollSecret>();
  const [showAddHostsModal, setShowAddHostsModal] = useState(false);
  const [
    showManageEnrollSecretsModal,
    setShowManageEnrollSecretsModal,
  ] = useState(false);
  const [showDeleteSecretModal, setShowDeleteSecretModal] = useState(false);
  const [showEnrollSecretModal, setShowEnrollSecretModal] = useState(false);
  const [showSecretEditorModal, setShowSecretEditorModal] = useState(false);
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const [isUpdatingTeams, setIsUpdatingTeams] = useState<boolean>(false);
  const { refetch: refetchMe } = useQuery(["me"], () => usersAPI.me(), {
    enabled: false,
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(available_teams);
    },
  });

  const {
    data: teams,
    isLoading: isLoadingTeams,
    refetch: refetchTeams,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ILoadTeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
      onSuccess: (responseTeams: ITeam[]) => {
        const findTeam = responseTeams.find(
          (team) => team.id === Number(routeParams.team_id)
        );

        if (findTeam) {
          setCurrentTeam(findTeam);
        } else {
          handlePageError({ status: 404 });
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  const {
    isLoading: isTeamSecretsLoading,
    data: teamSecrets,
    refetch: refetchTeamSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["team secrets", routeParams],
    () => {
      return enrollSecretsAPI.getTeamEnrollSecrets(teamIdFromURL);
    },
    {
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const navigateToNav = (i: number): void => {
    const navPath = teamDetailsSubNav[i].getPathname(teamIdFromURL);
    router.push(navPath);
  };

  const toggleAddHostsModal = useCallback(() => {
    setShowAddHostsModal(!showAddHostsModal);
  }, [showAddHostsModal, setShowAddHostsModal]);

  const toggleManageEnrollSecretsModal = useCallback(() => {
    setShowManageEnrollSecretsModal(!showManageEnrollSecretsModal);
  }, [showManageEnrollSecretsModal, setShowManageEnrollSecretsModal]);

  const toggleDeleteSecretModal = useCallback(() => {
    // open and closes delete modal
    setShowDeleteSecretModal(!showDeleteSecretModal);
    // open and closes main enroll secret modal
    setShowEnrollSecretModal(!showEnrollSecretModal);
  }, [
    setShowDeleteSecretModal,
    showDeleteSecretModal,
    setShowEnrollSecretModal,
    showEnrollSecretModal,
  ]);

  // this is called when we click add or edit
  const toggleSecretEditorModal = useCallback(() => {
    // open and closes add/edit modal
    setShowSecretEditorModal(!showSecretEditorModal);
    // open and closes main enroll secret modall
    setShowEnrollSecretModal(!showEnrollSecretModal);
  }, [
    setShowSecretEditorModal,
    showSecretEditorModal,
    setShowEnrollSecretModal,
    showEnrollSecretModal,
  ]);

  const toggleDeleteTeamModal = useCallback(() => {
    setShowDeleteTeamModal(!showDeleteTeamModal);
  }, [showDeleteTeamModal, setShowDeleteTeamModal]);

  const toggleEditTeamModal = useCallback(() => {
    setShowEditTeamModal(!showEditTeamModal);
    setBackendValidators({});
  }, [showEditTeamModal, setShowEditTeamModal, setBackendValidators]);

  const onSaveSecret = async (enrollSecretString: string) => {
    // Creates new list of secrets removing selected secret and adding new secret
    const currentSecrets = teamSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    if (enrollSecretString) {
      newSecrets.push({ secret: enrollSecretString });
    }

    try {
      await enrollSecretsAPI.modifyTeamEnrollSecrets(teamIdFromURL, newSecrets);
      refetchTeamSecrets();

      toggleSecretEditorModal();
      isPremiumTier && refetchTeams();
      renderFlash(
        "success",
        `Successfully ${selectedSecret ? "edited" : "added"} enroll secret.`
      );
    } catch (error) {
      console.error(error);
      renderFlash(
        "error",
        `Could not ${
          selectedSecret ? "edit" : "add"
        } enroll secret. Please try again.`
      );
    }
  };

  const onDeleteSecret = async () => {
    // create new list of secrets removing selected secret
    const currentSecrets = teamSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    try {
      await enrollSecretsAPI.modifyTeamEnrollSecrets(teamIdFromURL, newSecrets);
      refetchTeamSecrets();
      toggleDeleteSecretModal();
      refetchTeams();
      renderFlash("success", `Successfully deleted enroll secret.`);
    } catch (error) {
      console.error(error);
      renderFlash("error", "Could not delete enroll secret. Please try again.");
    }
  };

  const onDeleteSubmit = useCallback(async () => {
    if (!currentTeam) {
      return false;
    }

    setIsUpdatingTeams(true);

    try {
      await teamsAPI.destroy(currentTeam.id);
      renderFlash("success", "Team removed");
      return router.push(PATHS.ADMIN_TEAMS);
    } catch (response) {
      renderFlash("error", "Something went wrong removing the team");
      console.error(response);
      return false;
    } finally {
      toggleDeleteTeamModal();
      setIsUpdatingTeams(false);
    }
  }, [toggleDeleteTeamModal, currentTeam?.id]);

  const onEditSubmit = useCallback(
    async (formData: ITeamFormData) => {
      const updatedAttrs =
        currentTeam && generateUpdateData(currentTeam, formData);

      if (!currentTeam) {
        return false;
      }

      // no updates, so no need for a request.
      if (!updatedAttrs) {
        toggleEditTeamModal();
        return;
      }

      setIsUpdatingTeams(true);

      try {
        await teamsAPI.update(updatedAttrs, currentTeam.id);
        await teamsAPI.loadAll({ perPage: 500 });

        renderFlash(
          "success",
          `Successfully updated team name to ${updatedAttrs?.name}`
        );
        setBackendValidators({});
        refetchTeams();
        refetchMe();
        toggleEditTeamModal();
      } catch (response) {
        console.error(response);
        const errorObject = formatErrorResponse(response);

        if (errorObject.base.includes("Duplicate")) {
          setBackendValidators({
            name: "A team with this name already exists",
          });
        } else {
          renderFlash("error", "Could not create team. Please try again.");
          toggleEditTeamModal();
        }

        return false;
      } finally {
        setIsUpdatingTeams(false);
      }
    },
    [toggleEditTeamModal, currentTeam, setBackendValidators]
  );

  const handleTeamSelect = (teamId: number) => {
    const newSelectedTeam = find(teams, ["id", teamId]);
    const { ADMIN_TEAMS } = PATHS;

    const newRouteParams = {
      ...routeParams,
      team_id: newSelectedTeam ? newSelectedTeam.id : teamId,
    };

    const nextLocation = getNextLocationPath({
      pathPrefix: ADMIN_TEAMS,
      routeTemplate,
      routeParams: newRouteParams,
    });

    router.replace(`${nextLocation}/members`);

    setCurrentTeam(newSelectedTeam);
  };

  if (isLoadingTeams || isTeamSecretsLoading || currentTeam === undefined) {
    return (
      <div className={`${baseClass}__loading-spinner`}>
        <Spinner />
      </div>
    );
  }

  const hostCount = currentTeam.host_count;
  let hostsTotalDisplay: string | undefined;
  if (hostCount !== undefined) {
    hostsTotalDisplay =
      hostCount === 1 ? `${hostCount} host` : `${hostCount} hosts`;
  }

  const adminTeams = isGlobalAdmin
    ? availableTeams
    : availableTeams?.filter((t) => permissions.isTeamAdmin(currentUser, t.id));

  return (
    <MainContent className={baseClass}>
      <>
        <TabsWrapper>
          <>
            {isGlobalAdmin && (
              <Link
                to={PATHS.ADMIN_TEAMS}
                className={`${baseClass}__back-link`}
              >
                <img src={BackChevron} alt="back chevron" id="back-chevron" />
                <span>Back to teams</span>
              </Link>
            )}
          </>
          <div className={`${baseClass}__team-header`}>
            <div className={`${baseClass}__team-details`}>
              {adminTeams?.length === 1 ? (
                <h1>{currentTeam.name}</h1>
              ) : (
                <TeamsDropdown
                  selectedTeamId={toNumber(routeParams.team_id)}
                  currentUserTeams={adminTeams || []}
                  isDisabled={isLoadingTeams}
                  disableAll
                  onChange={(newSelectedValue: number) =>
                    handleTeamSelect(newSelectedValue)
                  }
                />
              )}
              {!!hostsTotalDisplay && (
                <span className={`${baseClass}__host-count`}>
                  {hostsTotalDisplay}
                </span>
              )}
            </div>
            <ActionButtons
              baseClass={baseClass}
              actions={[
                {
                  type: "primary",
                  label: "Add hosts",
                  onClick: toggleAddHostsModal,
                },
                {
                  type: "secondary",
                  label: "Manage enroll secrets",
                  buttonVariant: "text-icon",
                  icon: EyeIcon,
                  onClick: toggleManageEnrollSecretsModal,
                },
                {
                  type: "secondary",
                  label: "Edit team",
                  buttonVariant: "text-icon",
                  icon: PencilIcon,
                  onClick: toggleEditTeamModal,
                },
                {
                  type: "secondary",
                  label: "Delete team",
                  buttonVariant: "text-icon",
                  icon: TrashIcon,
                  hideAction: !isGlobalAdmin,
                  onClick: toggleDeleteTeamModal,
                },
              ]}
            />
          </div>
          <Tabs
            selectedIndex={getTabIndex(pathname, teamIdFromURL)}
            onSelect={(i) => navigateToNav(i)}
          >
            <TabList>
              {teamDetailsSubNav.map((navItem) => {
                // Bolding text when the tab is active causes a layout shift
                // so we add a hidden pseudo element with the same text string
                return (
                  <Tab key={navItem.name} data-text={navItem.name}>
                    {navItem.name}
                  </Tab>
                );
              })}
            </TabList>
          </Tabs>
        </TabsWrapper>
        {showAddHostsModal && (
          <AddHostsModal
            currentTeam={currentTeam}
            enrollSecret={teamSecrets?.[0]?.secret}
            isLoading={isLoadingTeams}
            // TODO: Currently, prepacked installers in Fleet Sandbox use the global enroll secret,
            // and Fleet Sandbox runs Fleet Free so explicitly setting isSandboxMode here is an
            // additional precaution/reminder to revisit this in connection with future changes.
            // See https://github.com/fleetdm/fleet/issues/4970#issuecomment-1187679407.
            isSandboxMode={false}
            onCancel={toggleAddHostsModal}
          />
        )}
        {showManageEnrollSecretsModal && (
          <EnrollSecretModal
            selectedTeam={teamIdFromURL}
            teams={teams || []}
            onReturnToApp={toggleManageEnrollSecretsModal}
            toggleSecretEditorModal={toggleSecretEditorModal}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
            setSelectedSecret={setSelectedSecret}
          />
        )}
        {showSecretEditorModal && (
          <SecretEditorModal
            selectedTeam={teamIdFromURL}
            teams={teams || []}
            onSaveSecret={onSaveSecret}
            toggleSecretEditorModal={toggleSecretEditorModal}
            selectedSecret={selectedSecret}
          />
        )}
        {showDeleteSecretModal && (
          <DeleteSecretModal
            onDeleteSecret={onDeleteSecret}
            selectedTeam={teamIdFromURL}
            teams={teams || []}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
          />
        )}
        {showDeleteTeamModal && (
          <DeleteTeamModal
            onCancel={toggleDeleteTeamModal}
            onSubmit={onDeleteSubmit}
            name={currentTeam.name}
            isUpdatingTeams={isUpdatingTeams}
          />
        )}
        {showEditTeamModal && (
          <EditTeamModal
            onCancel={toggleEditTeamModal}
            onSubmit={onEditSubmit}
            defaultName={currentTeam.name}
            backendValidators={backendValidators}
            isUpdatingTeams={isUpdatingTeams}
          />
        )}
        {children}
      </>
    </MainContent>
  );
};

export default TeamDetailsWrapper;
