import React, { useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";

import { getValidatedTeamId } from "utilities/helpers";

import { omit } from "lodash";

interface IControlsSubNavItem {
  name: string;
  pathname: string;
}

const controlsSubNav: IControlsSubNavItem[] = [
  {
    name: "macOS updates",
    pathname: PATHS.CONTROLS_MAC_UPDATES,
  },
  {
    name: "macOS settings",
    pathname: PATHS.CONTROLS_MAC_SETTINGS,
  },
];

interface IControlsWrapperProp {
  children: JSX.Element;
  location: any; // no type in react-router v3
  router: InjectedRouter; // v3
}

// Not sure what the below does, ask Rachel
const getTabIndex = (path: string): number => {
  return controlsSubNav.findIndex((navItem) => {
    // tab stays highlighted for paths that start with same pathname
    return path.startsWith(navItem.pathname);
  });
};

const baseClass = "controls-wrapper";

const ControlsWrapper = ({
  children,
  location,
  router,
}: IControlsWrapperProp): JSX.Element => {
  const queryParams = location.query;
  const {
    currentUser,
    isOnGlobalTeam,
    availableTeams,
    currentTeam,
    isPremiumTier,
    setCurrentTeam,
  } = useContext(AppContext);

  const navigateToNav = (i: number): void => {
    const navPath = controlsSubNav[i].pathname;
    router.push(navPath);
  };

  const handleTeamSelect = (teamId: number) => {
    const { CONTROLS } = PATHS;

    const teamIdParam = getValidatedTeamId(
      availableTeams || [],
      teamId,
      currentUser,
      isOnGlobalTeam ?? false
    );

    const slimmerParams = omit(queryParams, ["team_id"]);

    const newQueryParams = !teamIdParam
      ? slimmerParams
      : Object.assign(slimmerParams, { team_id: teamIdParam });

    const nextLocation = getNextLocationPath({
      pathPrefix: MANAGE_HOSTS,
      routeTemplate,
      routeParams,
      queryParams: newQueryParams,
    });

    handleResetPageIndex();
    router.replace(nextLocation);
    const selectedTeam = find(availableTeams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const renderHeader = () => (
    <div className={`${baseClass}__header`}>
      <div className={`${baseClass}__text`}>
        <div className={`${baseClass}__title`}>
          {isPremiumTier ? (
            <TeamsDropdown
              currentUserTeams={availableTeams || []}
              selectedTeamId={currentTeam?.id}
              onChange={(newSelectedValue: number) =>
                handleTeamSelect(newSelectedValue)
              }
            />
          ) : (
            <h1>Controls</h1>
          )}
          {/* {isPremiumTier &&
            availableTeams &&
            (availableTeams.length > 1 || isOnGlobalTeam) &&
            renderTeamsFilterDropdown()}
          {isPremiumTier &&
            !isOnGlobalTeam &&
            availableTeams &&
            availableTeams.length === 1 && <h1>{availableTeams[0].name}</h1>} */}
        </div>
      </div>
    </div>
  );

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>{renderHeader()}</div>
        {isPremiumTier ? (
          <div>
            <TabsWrapper>
              <Tabs
                selectedIndex={getTabIndex(location.pathname)}
                onSelect={(i) => navigateToNav(i)}
              >
                <TabList>
                  {controlsSubNav.map((navItem) => {
                    return (
                      <Tab key={navItem.name} data-text={navItem.name}>
                        {navItem.name}
                      </Tab>
                    );
                  })}
                </TabList>
              </Tabs>
            </TabsWrapper>
            {children}
          </div>
        ) : (
          <>
            <hr />
            <h1> Buy Premium</h1>
          </>
        )}
      </div>
    </MainContent>
  );
};

export default ControlsWrapper;
