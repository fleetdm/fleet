import React, { useState, useEffect, useCallback, useContext } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { InjectedRouter, Link, RouteProps } from "react-router";
import { push } from "react-router-redux";
import { Tab, TabList, Tabs } from "react-tabs";
import { find, toNumber } from "lodash";
import classnames from "classnames";

import PATHS from "router/paths";
import { ITeam, ITeamSummary } from "interfaces/team";
import { AppContext } from "context/app";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import teamsAPI from "services/entities/teams";
import enrollSecretsAPI from "services/entities/enroll_secret";
import teamActions from "redux/nodes/entities/teams/actions";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import sortUtils from "utilities/sort";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TabsWrapper from "components/TabsWrapper";
import TeamsDropdown from "components/TeamsDropdown";
import { getNextLocationPath } from "pages/admin/UserManagementPage/helpers/userManagementHelpers";
import DeleteTeamModal from "../components/DeleteTeamModal";
import EditTeamModal from "../components/EditTeamModal";
import { IEditTeamFormData } from "../components/EditTeamModal/EditTeamModal";
import DeleteSecretModal from "../../../../components/DeleteSecretModal";
import SecretEditorModal from "../../../../components/SecretEditorModal";
import GenerateInstallerModal from "../../../../components/GenerateInstallerModal";
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

interface ITeamsResponse {
  teams: ITeam[];
}

interface ITeamDetailsPageProps {
  children: JSX.Element;
  params: {
    team_id: number;
  };
  location: {
    pathname: string;
  };
  route: RouteProps;
  router: InjectedRouter;
}

const generateUpdateData = (
  currentTeam: ITeamSummary,
  formData: IEditTeamFormData
): IEditTeamFormData | null => {
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
  const dispatch = useDispatch();

  const {
    currentUser,
    isGlobalAdmin,
    currentTeam,
    isOnGlobalTeam,
    isPremiumTier,
    setCurrentTeam,
  } = useContext(AppContext);

  const userTeams = currentUser?.teams || [];
  const routeTemplate = route && route.path ? route.path : "";

  const [selectedSecret, setSelectedSecret] = useState<IEnrollSecret>();
  const [showGenerateInstallerModal, setShowGenerateInstallerModal] = useState(
    false
  );
  const [
    showManageEnrollSecretsModal,
    setShowManageEnrollSecretsModal,
  ] = useState(false);
  const [showDeleteSecretModal, setShowDeleteSecretModal] = useState(false);
  const [showEnrollSecretModal, setShowEnrollSecretModal] = useState(false);
  const [showSecretEditorModal, setShowSecretEditorModal] = useState(false);
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);

  const {
    data: teams,
    isLoading: isLoadingTeams,
    refetch: refetchTeams,
  } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ITeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
      onSuccess: (responseTeams: ITeam[]) => {
        const findTeam = responseTeams.find(
          (team) => team.id === Number(routeParams.team_id)
        );
        setCurrentTeam(findTeam);
      },
    }
  );

  const {
    isLoading: isTeamSecretsLoading,
    data: teamSecrets,
    refetch: refetchTeamSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["team secrets", routeParams],
    () => {
      return enrollSecretsAPI.getTeamEnrollSecrets(routeParams.team_id);
    },
    {
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const navigateToNav = (i: number): void => {
    const navPath = teamDetailsSubNav[i].getPathname(routeParams.team_id);
    dispatch(push(navPath));
  };

  useEffect(() => {
    dispatch(teamActions.loadAll({ perPage: 500 }));
  }, [dispatch]);

  const [teamMenuIsOpen, setTeamMenuIsOpen] = useState<boolean>(false);

  const toggleGenerateInstallerModal = useCallback(() => {
    setShowGenerateInstallerModal(!showGenerateInstallerModal);
  }, [showGenerateInstallerModal, setShowGenerateInstallerModal]);

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
  }, [showEditTeamModal, setShowEditTeamModal]);

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
      await enrollSecretsAPI.modifyTeamEnrollSecrets(
        routeParams.team_id,
        newSecrets
      );
      refetchTeamSecrets();

      toggleSecretEditorModal();
      isPremiumTier && refetchTeams();
      dispatch(
        renderFlash(
          "success",
          `Successfully ${selectedSecret ? "edited" : "added"} enroll secret.`
        )
      );
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash(
          "error",
          `Could not ${
            selectedSecret ? "edit" : "add"
          } enroll secret. Please try again.`
        )
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
      await enrollSecretsAPI.modifyTeamEnrollSecrets(
        routeParams.team_id,
        newSecrets
      );
      refetchTeamSecrets();
      toggleDeleteSecretModal();
      refetchTeams();
      dispatch(renderFlash("success", `Successfully deleted enroll secret.`));
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash(
          "error",
          "Could not delete enroll secret. Please try again."
        )
      );
    }
  };

  const onDeleteSubmit = useCallback(() => {
    dispatch(teamActions.destroy(currentTeam?.id))
      .then(() => {
        dispatch(renderFlash("success", "Team removed"));
        dispatch(push(PATHS.ADMIN_TEAMS));
        // TODO: error handling
      })
      .catch(() => null);
    toggleDeleteTeamModal();
  }, [dispatch, toggleDeleteTeamModal, currentTeam?.id]);

  const onEditSubmit = useCallback(
    (formData: IEditTeamFormData) => {
      const updatedAttrs =
        currentTeam && generateUpdateData(currentTeam, formData);
      // no updates, so no need for a request.
      if (updatedAttrs === null) {
        toggleEditTeamModal();
        return;
      }
      dispatch(teamActions.update(currentTeam?.id, updatedAttrs))
        .then(() => {
          dispatch(teamActions.loadAll({ perPage: 500 }));
          dispatch(renderFlash("success", "Team updated"));
          // TODO: error handling
        })
        .catch(() => null);
      toggleEditTeamModal();
    },
    [dispatch, toggleEditTeamModal, currentTeam]
  );

  const handleTeamSelect = (teamId: number) => {
    const selectedTeam = find(teams, ["id", teamId]);
    const { ADMIN_TEAMS } = PATHS;

    const newRouteParams = {
      ...routeParams,
      team_id: selectedTeam ? selectedTeam.id : teamId,
    };

    const nextLocation = getNextLocationPath({
      pathPrefix: ADMIN_TEAMS,
      routeTemplate,
      routeParams: newRouteParams,
    });

    router.replace(`${nextLocation}/members`);

    setCurrentTeam(selectedTeam);
  };

  const handleTeamMenuOpen = () => {
    setTeamMenuIsOpen(true);
  };

  const handleTeamMenuClose = () => {
    setTeamMenuIsOpen(false);
  };

  const teamWrapperClasses = classnames(baseClass, {
    "team-select-open": teamMenuIsOpen,
    "team-settings": !isOnGlobalTeam,
  });

  if (isLoadingTeams || isTeamSecretsLoading || currentTeam === undefined) {
    return (
      <div className={`${baseClass}__loading-spinner`}>
        <Spinner />
      </div>
    );
  }

  // const hostsCount = currentTeam.host_count;
  const hostsCount = teams?.length || 1;
  const hostsTotalDisplay = hostsCount === 1 ? "1 host" : `${hostsCount} hosts`;
  const userAdminTeams = userTeams.filter(
    (thisTeam) => thisTeam.role === "admin"
  );
  const adminTeams = isGlobalAdmin ? teams : userAdminTeams;

  return (
    <div className={teamWrapperClasses}>
      <TabsWrapper>
        <>
          {isGlobalAdmin && (
            <Link to={PATHS.ADMIN_TEAMS} className={`${baseClass}__back-link`}>
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
                onOpen={handleTeamMenuOpen}
                onClose={handleTeamMenuClose}
              />
            )}
            <span className={`${baseClass}__host-count`}>
              {hostsTotalDisplay}
            </span>
          </div>
          <div className={`${baseClass}__team-actions`}>
            <Button onClick={toggleGenerateInstallerModal}>
              Generate installer
            </Button>
            <Button
              onClick={toggleManageEnrollSecretsModal}
              variant={"text-icon"}
            >
              <>
                Manage enroll secrets{" "}
                <img src={EyeIcon} alt="Manage enroll secrets icon" />
              </>
            </Button>
            <Button onClick={toggleEditTeamModal} variant={"text-icon"}>
              <>
                Edit team <img src={PencilIcon} alt="Edit team icon" />
              </>
            </Button>
            {isGlobalAdmin && (
              <Button onClick={toggleDeleteTeamModal} variant={"text-icon"}>
                <>
                  Delete team <img src={TrashIcon} alt="Delete team icon" />
                </>
              </Button>
            )}
          </div>
        </div>
        <Tabs
          selectedIndex={getTabIndex(pathname, routeParams.team_id)}
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
      {showGenerateInstallerModal && (
        <GenerateInstallerModal
          onCancel={toggleGenerateInstallerModal}
          selectedTeam={{
            name: currentTeam.name,
            secrets: teamSecrets || null,
          }}
        />
      )}
      {showManageEnrollSecretsModal && (
        <EnrollSecretModal
          selectedTeam={routeParams.team_id}
          teams={teams || []}
          onReturnToApp={toggleManageEnrollSecretsModal}
          toggleSecretEditorModal={toggleSecretEditorModal}
          toggleDeleteSecretModal={toggleDeleteSecretModal}
          setSelectedSecret={setSelectedSecret}
        />
      )}
      {showSecretEditorModal && (
        <SecretEditorModal
          selectedTeam={routeParams.team_id}
          teams={teams || []}
          onSaveSecret={onSaveSecret}
          toggleSecretEditorModal={toggleSecretEditorModal}
          selectedSecret={selectedSecret}
        />
      )}
      {showDeleteSecretModal && (
        <DeleteSecretModal
          onDeleteSecret={onDeleteSecret}
          selectedTeam={routeParams.team_id}
          teams={teams || []}
          toggleDeleteSecretModal={toggleDeleteSecretModal}
        />
      )}
      {showDeleteTeamModal && (
        <DeleteTeamModal
          onCancel={toggleDeleteTeamModal}
          onSubmit={onDeleteSubmit}
          name={currentTeam.name}
        />
      )}
      {showEditTeamModal && (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          onSubmit={onEditSubmit}
          defaultName={currentTeam.name}
        />
      )}
      {children}
    </div>
  );
};

export default TeamDetailsWrapper;
