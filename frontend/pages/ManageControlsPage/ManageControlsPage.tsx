import React, { useCallback, useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import useTeamIdParam from "hooks/useTeamIdParam";

import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";

interface IControlsSubNavItem {
  name: string;
  pathname: string;
}

const controlsSubNav: IControlsSubNavItem[] = [
  {
    name: "OS updates",
    pathname: PATHS.CONTROLS_OS_UPDATES,
  },
  {
    name: "OS settings",
    pathname: PATHS.CONTROLS_OS_SETTINGS,
  },
  {
    name: "Setup experience",
    pathname: PATHS.CONTROLS_SETUP_EXPERIENCE,
  },
];

interface IManageControlsPageProps {
  children: JSX.Element;
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: {
      team_id?: string;
    };
  };
  router: InjectedRouter; // v3
}

const getTabIndex = (path: string): number => {
  return controlsSubNav.findIndex((navItem) => {
    // tab stays highlighted for paths that start with same pathname
    return path.startsWith(navItem.pathname);
  });
};

const baseClass = "manage-controls-page";

const ManageControlsPage = ({
  // TODO(sarah): decide on pattern to pass team id to subcomponents.
  // using children makes it difficult to centralize page-level control
  // over team id param
  children,
  location,
  router,
}: IManageControlsPageProps): JSX.Element => {
  const {
    isFreeTier,
    isOnGlobalTeam,
    isPremiumTier,
    isSandboxMode,
  } = useContext(AppContext);

  const {
    currentTeamId,
    userTeams,
    teamIdForApi,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: true,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: false,
      observer_plus: false,
    },
  });

  const navigateToNav = useCallback(
    (i: number): void => {
      const navPath = controlsSubNav[i].pathname;
      router.replace(
        navPath.concat(location?.search || "").concat(location?.hash || "")
      );
    },
    [location, router]
  );

  const renderBody = () => {
    return (
      <div>
        <TabsWrapper>
          <Tabs
            selectedIndex={getTabIndex(location?.pathname || "")}
            onSelect={navigateToNav}
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
        {React.cloneElement(children, { teamIdForApi })}
      </div>
    );
  };

  return (
    <MainContent>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header-wrap`}>
            <div className={`${baseClass}__header`}>
              <div className={`${baseClass}__text`}>
                <div className={`${baseClass}__title`}>
                  {isFreeTier && <h1>Controls</h1>}
                  {isPremiumTier &&
                    userTeams &&
                    (userTeams.length > 1 || isOnGlobalTeam) && (
                      <TeamsDropdown
                        currentUserTeams={userTeams}
                        selectedTeamId={currentTeamId}
                        onChange={handleTeamChange}
                        includeAll={false}
                        includeNoTeams
                        isSandboxMode={isSandboxMode}
                      />
                    )}
                  {isPremiumTier &&
                    !isOnGlobalTeam &&
                    userTeams &&
                    userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
                </div>
              </div>
            </div>
          </div>
        </div>
        {renderBody()}
      </div>
    </MainContent>
  );
};

export default ManageControlsPage;
