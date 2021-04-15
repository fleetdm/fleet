import React, { useState, useEffect } from "react";
import { useSelector, useDispatch } from "react-redux";
import { Link } from "react-router";
import { push } from "react-router-redux";
import { Tab, TabList, Tabs } from "react-tabs";

import PATHS from "router/paths";
import { ITeam } from "interfaces/team";
import teamActions from "redux/nodes/entities/teams/actions";
import Button from "../../../../components/buttons/Button";

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

interface ITeamDetailsPageProps {
  children: JSX.Element;
  params: {
    team_id: number;
  };
  location: {
    pathname: string;
  };
}

const getTabIndex = (path: string, teamId: number): number => {
  return teamDetailsSubNav.findIndex((navItem) => {
    return navItem.getPathname(teamId).includes(path);
  });
};

const TeamDetailsPage = (props: ITeamDetailsPageProps): JSX.Element => {
  const {
    children,
    location: { pathname },
    params: { team_id },
  } = props;

  const dispatch = useDispatch();

  const navigateToNav = (i: number): void => {
    const navPath = teamDetailsSubNav[i].getPathname(team_id);
    dispatch(push(navPath));
  };

  useEffect(() => {
    // dispatch(teamActions.load());
  }, []);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__nav-header`}>
        <Link className={`${baseClass}__back-link`} to={PATHS.ADMIN_TEAMS}>
          Back to teams
        </Link>
        <div className={`${baseClass}__team-header`}>
          <div className={`${baseClass}__team-details`}>
            <h1>Test Team 2</h1>
            <span className={`${baseClass}__host-count`}>0 hosts</span>
          </div>
          <div className={`${baseClass}__team-actions`}>
            <Button onClick={() => console.log("click")}>Add hosts</Button>
            <Button onClick={() => console.log("click")}>Edit team</Button>
            <Button onClick={() => console.log("click")}>Delete team</Button>
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
      {children}
    </div>
  );
};

export default TeamDetailsPage;
