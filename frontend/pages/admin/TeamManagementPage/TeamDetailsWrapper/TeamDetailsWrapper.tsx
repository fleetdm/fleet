import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter } from "react-router";
import { Tab, TabList, Tabs } from "react-tabs";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import useTeamIdParam from "hooks/useTeamIdParam";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { ITeam, ITeamSummary } from "interfaces/team";
import PATHS from "router/paths";
import enrollSecretsAPI from "services/entities/enroll_secret";
import teamsAPI, {
  ILoadTeamsResponse,
  ITeamFormData,
} from "services/entities/teams";
import usersAPI, { IGetMeResponse } from "services/entities/users";
import formatErrorResponse from "utilities/format_error_response";
import sortUtils from "utilities/sort";

import ActionButtons from "components/buttons/ActionButtons/ActionButtons";
import Spinner from "components/Spinner";
import TabsWrapper from "components/TabsWrapper";
import BackLink from "components/BackLink";
import TeamsDropdown from "components/TeamsDropdown";
import MainContent from "components/MainContent";
import DeleteTeamModal from "../components/DeleteTeamModal";
import RenameTeamModal from "../components/RenameTeamModal";
import DeleteSecretModal from "../../../../components/EnrollSecrets/DeleteSecretModal";
import SecretEditorModal from "../../../../components/EnrollSecrets/SecretEditorModal";
import AddHostsModal from "../../../../components/AddHostsModal";
import EnrollSecretModal from "../../../../components/EnrollSecrets/EnrollSecretModal";

const baseClass = "team-details";

interface ITeamDetailsSubNavItem {
  name: string;
  getPathname: (id?: number) => string;
}

const teamDetailsSubNav: ITeamDetailsSubNavItem[] = [
  {
    name: "Users",
    getPathname: PATHS.TEAM_DETAILS_USERS,
  },
  {
    name: "Agent options",
    getPathname: PATHS.TEAM_DETAILS_OPTIONS,
  },
  {
    name: "Settings",
    getPathname: PATHS.TEAM_DETAILS_SETTINGS,
  },
];

interface ITeamDetailsPageProps {
  children: JSX.Element;
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: { team_id?: string };
  };
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
  router,
  children,
  location,
}: ITeamDetailsPageProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const handlePageError = useErrorHandler();
  const {
    isGlobalAdmin,
    isPremiumTier,
    setAvailableTeams,
    setCurrentUser,
  } = useContext(AppContext);

  const {
    currentTeamId,
    currentTeamName,
    isAnyTeamSelected,
    isRouteOk,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

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
  const [showRenameTeamModal, setShowRenameTeamModal] = useState(false);
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const [isUpdatingTeams, setIsUpdatingTeams] = useState(false);
  const [isUpdatingSecret, setIsUpdatingSecret] = useState(false);

  const { refetch: refetchMe } = useQuery(["me"], () => usersAPI.me(), {
    enabled: false,
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
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
      enabled: isRouteOk,
      select: (data: ILoadTeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
      onSuccess: (responseTeams: ITeam[]) => {
        if (!responseTeams?.find((team) => team.id === teamIdForApi)) {
          handlePageError({ status: 404 });
        }
      },
      onError: (error) => handlePageError(error),
    }
  );
  const currentTeamDetails = teams?.find((team) => team.id === teamIdForApi);

  const {
    isLoading: isTeamSecretsLoading,
    data: teamSecrets,
    refetch: refetchTeamSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["team secrets", teamIdForApi],
    () => {
      return enrollSecretsAPI.getTeamEnrollSecrets(teamIdForApi);
    },
    {
      enabled: isRouteOk,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const navigateToNav = (i: number): void => {
    const navPath = teamDetailsSubNav[i].getPathname(teamIdForApi);
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

  const toggleRenameTeamModal = useCallback(() => {
    setShowRenameTeamModal(!showRenameTeamModal);
    setBackendValidators({});
  }, [showRenameTeamModal, setShowRenameTeamModal, setBackendValidators]);

  const onSaveSecret = async (enrollSecretString: string) => {
    // Creates new list of secrets removing selected secret and adding new secret
    const currentSecrets = teamSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    if (enrollSecretString) {
      newSecrets.push({ secret: enrollSecretString });
    }
    setIsUpdatingSecret(true);
    try {
      await enrollSecretsAPI.modifyTeamEnrollSecrets(teamIdForApi, newSecrets);
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
    } finally {
      setIsUpdatingSecret(false);
    }
  };

  const onDeleteSecret = async () => {
    // create new list of secrets removing selected secret
    const currentSecrets = teamSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );
    setIsUpdatingSecret(true);
    try {
      await enrollSecretsAPI.modifyTeamEnrollSecrets(teamIdForApi, newSecrets);
      refetchTeamSecrets();
      toggleDeleteSecretModal();
      refetchTeams();
      renderFlash("success", `Successfully deleted enroll secret.`);
    } catch (error) {
      console.error(error);
      renderFlash("error", "Could not delete enroll secret. Please try again.");
    } finally {
      setIsUpdatingSecret(false);
    }
  };

  const onDeleteSubmit = useCallback(async () => {
    if (!teamIdForApi) {
      return false;
    }

    setIsUpdatingTeams(true);

    try {
      await teamsAPI.destroy(teamIdForApi);
      router.push(PATHS.ADMIN_TEAMS);
      renderFlash("success", "Team removed");
    } catch (response) {
      renderFlash("error", "Something went wrong removing the team");
      console.error(response);
    } finally {
      toggleDeleteTeamModal();
      setIsUpdatingTeams(false);
    }
  }, [teamIdForApi, renderFlash, router, toggleDeleteTeamModal]);

  const onEditSubmit = useCallback(
    async (formData: ITeamFormData) => {
      if (!currentTeamDetails) {
        return;
      }
      const updatedAttrs = generateUpdateData(currentTeamDetails, formData);
      // no updates, so no need for a request.
      if (!updatedAttrs) {
        toggleRenameTeamModal();
        return;
      }

      setIsUpdatingTeams(true);
      try {
        await teamsAPI.update(updatedAttrs, teamIdForApi);
        renderFlash(
          "success",
          `Successfully updated team name to ${updatedAttrs?.name}`
        );
        setBackendValidators({});
        refetchTeams();
        refetchMe();
        toggleRenameTeamModal();
      } catch (response) {
        console.error(response);
        const errorObject = formatErrorResponse(response);
        if (errorObject.base.includes("Duplicate")) {
          setBackendValidators({
            name: "A team with this name already exists",
          });
        } else if (errorObject.base.includes("all teams")) {
          setBackendValidators({
            name: `"All teams" is a reserved team name. Please try another name.`,
          });
        } else if (errorObject.base.includes("no team")) {
          setBackendValidators({
            name: `"No team" is a reserved team name. Please try another name.`,
          });
        } else {
          renderFlash("error", "Could not create team. Please try again.");
        }
      } finally {
        setIsUpdatingTeams(false);
      }
    },
    [
      currentTeamDetails,
      toggleRenameTeamModal,
      teamIdForApi,
      renderFlash,
      refetchTeams,
      refetchMe,
    ]
  );

  if (
    !isRouteOk ||
    isLoadingTeams ||
    isTeamSecretsLoading ||
    !userTeams?.length ||
    currentTeamDetails === undefined
  ) {
    return (
      <div className={`${baseClass}__loading-spinner`}>
        <Spinner />
      </div>
    );
  }

  const hostCount = currentTeamDetails.host_count;
  let hostsTotalDisplay: string | undefined;
  if (hostCount !== undefined) {
    hostsTotalDisplay =
      hostCount === 1 ? `${hostCount} host` : `${hostCount} hosts`;
  }

  return (
    <MainContent className={baseClass}>
      <>
        <TabsWrapper>
          {isGlobalAdmin ? (
            <div className={`${baseClass}__header-links`}>
              <BackLink text="Back to teams" path={PATHS.ADMIN_TEAMS} />
            </div>
          ) : (
            <></>
          )}
          <div className={`${baseClass}__team-header`}>
            <div className={`${baseClass}__team-details`}>
              {userTeams?.length === 1 ? (
                <h1>{currentTeamDetails.name}</h1>
              ) : (
                <TeamsDropdown
                  selectedTeamId={currentTeamId}
                  currentUserTeams={userTeams || []}
                  isDisabled={isLoadingTeams}
                  includeAll={false}
                  onChange={handleTeamChange}
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
                  iconSvg: "eye",
                  onClick: toggleManageEnrollSecretsModal,
                },
                {
                  type: "secondary",
                  label: "Rename team",
                  buttonVariant: "text-icon",
                  iconSvg: "pencil",
                  onClick: toggleRenameTeamModal,
                },
                {
                  type: "secondary",
                  label: "Delete team",
                  buttonVariant: "text-icon",
                  iconSvg: "trash",
                  hideAction: !isGlobalAdmin,
                  onClick: toggleDeleteTeamModal,
                },
              ]}
            />
          </div>
          <Tabs
            selectedIndex={getTabIndex(
              location.pathname,
              currentTeamDetails.id
            )}
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
            currentTeamName={currentTeamName}
            enrollSecret={teamSecrets?.[0]?.secret}
            isAnyTeamSelected={isAnyTeamSelected}
            isLoading={isLoadingTeams}
            onCancel={toggleAddHostsModal}
            openEnrollSecretModal={toggleManageEnrollSecretsModal}
          />
        )}
        {showManageEnrollSecretsModal && (
          <EnrollSecretModal
            selectedTeam={teamIdForApi || 0} // TODO: confirm teamIdForApi vs currentTeamId throughout
            teams={teams || []} // TODO: confirm teams vs available teams throughout
            onReturnToApp={toggleManageEnrollSecretsModal}
            toggleSecretEditorModal={toggleSecretEditorModal}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
            setSelectedSecret={setSelectedSecret}
          />
        )}
        {showSecretEditorModal && (
          <SecretEditorModal
            selectedTeam={currentTeamDetails.id}
            teams={teams || []}
            onSaveSecret={onSaveSecret}
            toggleSecretEditorModal={toggleSecretEditorModal}
            selectedSecret={selectedSecret}
            isUpdatingSecret={isUpdatingSecret}
          />
        )}
        {showDeleteSecretModal && (
          <DeleteSecretModal
            onDeleteSecret={onDeleteSecret}
            selectedTeam={teamIdForApi || 0}
            teams={teams || []}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
            isUpdatingSecret={isUpdatingSecret}
          />
        )}
        {showDeleteTeamModal && (
          <DeleteTeamModal
            onCancel={toggleDeleteTeamModal}
            onSubmit={onDeleteSubmit}
            name={currentTeamDetails.name}
            isUpdatingTeams={isUpdatingTeams}
          />
        )}
        {showRenameTeamModal && (
          <RenameTeamModal
            onCancel={toggleRenameTeamModal}
            onSubmit={onEditSubmit}
            defaultName={currentTeamDetails.name}
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
