import React, { useState, useEffect, useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import { Link } from "react-router";
import { push } from "react-router-redux";
import { Tab, TabList, Tabs } from "react-tabs";

import PATHS from "router/paths";
import { ITeam } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import Spinner from "components/loaders/Spinner";
import Button from "components/buttons/Button";
import DeleteTeamModal from "../components/DeleteTeamModal";
import EditTeamModal from "../components/EditTeamModal";
import { IEditTeamFormData } from "../components/EditTeamModal/EditTeamModal";
import AddHostsRedirectModal from "./components/AddHostsModal/AddHostsRedirectModal";

import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";
import PencilIcon from "../../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../../assets/images/icon-trash-14x14@2x.png";

const baseClass = "team-details";

interface ITeamDetailsSubNavItem {
  name: string;
  getPathname: (id: number) => string;
}

const teamDetailsSubNav: ITeamDetailsSubNavItem[] = [
  {
    name: "Member",
    getPathname: PATHS.TEAM_DETAILS_MEMBERS,
  },
  {
    name: "Agent options",
    getPathname: PATHS.TEAM_DETAILS_OPTIONS,
  },
];

interface IRootState {
  entities: {
    teams: {
      loading: boolean;
      data: { [id: number]: ITeam };
    };
  };
}

interface ITeamDetailsPageProps {
  children: JSX.Element;
  params: {
    team_id: number;
  };
  location: {
    pathname: string;
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

const getTabIndex = (path: string, teamId: number): number => {
  return teamDetailsSubNav.findIndex((navItem) => {
    return navItem.getPathname(teamId).includes(path);
  });
};

const TeamDetailsWrapper = (props: ITeamDetailsPageProps): JSX.Element => {
  const {
    children,
    location: { pathname },
    params: { team_id },
  } = props;

  const isLoadingTeams = useSelector(
    (state: IRootState) => state.entities.teams.loading
  );
  const team = useSelector((state: IRootState) => {
    return state.entities.teams.data[team_id];
  });

  const [showAddHostsRedirectModal, setShowAddHostsRedirectModal] = useState(
    false
  );
  const [showDeleteTeamModal, setShowDeleteTeamModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);

  const dispatch = useDispatch();

  const navigateToNav = (i: number): void => {
    const navPath = teamDetailsSubNav[i].getPathname(team_id);
    dispatch(push(navPath));
  };

  useEffect(() => {
    dispatch(teamActions.loadAll({ perPage: 500 }));
  }, [dispatch]);

  const toggleAddHostsRedirectModal = useCallback(() => {
    setShowAddHostsRedirectModal(!showAddHostsRedirectModal);
  }, [showAddHostsRedirectModal, setShowAddHostsRedirectModal]);

  const toggleDeleteTeamModal = useCallback(() => {
    setShowDeleteTeamModal(!showDeleteTeamModal);
  }, [showDeleteTeamModal, setShowDeleteTeamModal]);

  const toggleEditTeamModal = useCallback(() => {
    setShowEditTeamModal(!showEditTeamModal);
  }, [showEditTeamModal, setShowEditTeamModal]);

  const onAddHostsRedirectClick = useCallback(() => {
    dispatch(push(PATHS.MANAGE_HOSTS));
  }, [dispatch]);

  const onDeleteSubmit = useCallback(() => {
    dispatch(teamActions.destroy(team?.id))
      .then(() => {
        dispatch(renderFlash("success", "Team removed"));
        dispatch(push(PATHS.ADMIN_TEAMS));
        // TODO: error handling
      })
      .catch(() => null);
    toggleDeleteTeamModal();
  }, [dispatch, toggleDeleteTeamModal, team?.id]);

  const onEditSubmit = useCallback(
    (formData: IEditTeamFormData) => {
      const updatedAttrs = generateUpdateData(team, formData);
      // no updates, so no need for a request.
      if (updatedAttrs === null) {
        toggleEditTeamModal();
        return;
      }
      dispatch(teamActions.update(team?.id, updatedAttrs))
        .then(() => {
          dispatch(teamActions.loadAll({ perPage: 500 }));
          dispatch(renderFlash("success", "Team updated"));
          // TODO: error handling
        })
        .catch(() => null);
      toggleEditTeamModal();
    },
    [dispatch, toggleEditTeamModal, team]
  );

  if (isLoadingTeams || team === undefined) {
    return (
      <div className={`${baseClass}__loading-spinner`}>
        <Spinner />
      </div>
    );
  }
  const hostsCount = team.host_count;
  const hostsTotalDisplay = hostsCount === 1 ? "1 host" : `${hostsCount} hosts`;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__nav-header`}>
        <Link to={PATHS.ADMIN_TEAMS} className={`${baseClass}__back-link`}>
          <img src={BackChevron} alt="back chevron" id="back-chevron" />
          <span>Back to teams</span>
        </Link>
        <div className={`${baseClass}__team-header`}>
          <div className={`${baseClass}__team-details`}>
            <h1>{team.name}</h1>
            <span className={`${baseClass}__host-count`}>
              {hostsTotalDisplay}
            </span>
          </div>
          <div className={`${baseClass}__team-actions`}>
            <Button onClick={toggleAddHostsRedirectModal}>Add hosts</Button>
            <Button onClick={toggleEditTeamModal} variant={"text-icon"}>
              <>
                <img src={PencilIcon} alt="Edit team icon" />
                Edit team
              </>
            </Button>
            <Button onClick={toggleDeleteTeamModal} variant={"text-icon"}>
              <>
                <img src={TrashIcon} alt="Delete team icon" />
                Delete team
              </>
            </Button>
          </div>
        </div>
        <Tabs
          selectedIndex={getTabIndex(pathname, team_id)}
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
      </div>
      {showAddHostsRedirectModal ? (
        <AddHostsRedirectModal
          onCancel={toggleAddHostsRedirectModal}
          onSubmit={onAddHostsRedirectClick}
        />
      ) : null}
      {showDeleteTeamModal ? (
        <DeleteTeamModal
          onCancel={toggleDeleteTeamModal}
          onSubmit={onDeleteSubmit}
          name={team.name}
        />
      ) : null}
      {showEditTeamModal ? (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          onSubmit={onEditSubmit}
          defaultName={team.name}
        />
      ) : null}
      {children}
    </div>
  );
};

export default TeamDetailsWrapper;
