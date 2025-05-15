import React, { useCallback, useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import useTeamIdParam from "hooks/useTeamIdParam";

import TabNav from "components/TabNav";
import TabText from "components/TabText";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";
import { parseOSUpdatesCurrentVersionsQueryParams } from "./OSUpdates/components/CurrentVersionSection/CurrentVersionSection";

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
  {
    name: "Scripts",
    pathname: PATHS.CONTROLS_SCRIPTS,
  },
];

const subNavQueryParams = ["page", "order_key", "order_direction"] as const;

interface IManageControlsPageProps {
  children: JSX.Element;
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: {
      team_id?: string;
      page?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
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
  const page = parseInt(location?.query?.page || "", 10) || 0;

  const { config, isOnGlobalTeam, isPremiumTier } = useContext(AppContext);

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
      // remove query params related to the prior tab
      const newParams = new URLSearchParams(location?.search);
      subNavQueryParams.forEach((p) => newParams.delete(p));
      const newQuery = newParams.toString();

      router.replace(
        navPath
          .concat(newQuery ? `?${newQuery}` : "")
          .concat(location?.hash || "")
      );
    },
    [location, router]
  );

  const renderBody = () => {
    return (
      <div>
        <TabNav>
          <Tabs
            selectedIndex={getTabIndex(location?.pathname || "")}
            onSelect={navigateToNav}
          >
            <TabList>
              {controlsSubNav.map((navItem) => {
                return (
                  <Tab key={navItem.name} data-text={navItem.name}>
                    <TabText>{navItem.name}</TabText>
                  </Tab>
                );
              })}
            </TabList>
          </Tabs>
        </TabNav>
        {React.cloneElement(children, {
          teamIdForApi,
          currentPage: page,
          queryParams: parseOSUpdatesCurrentVersionsQueryParams(location.query),
        })}
      </div>
    );
  };

  const renderHeaderContent = () => {
    if (isPremiumTier && !config?.partnerships?.enable_primo && userTeams) {
      if (userTeams.length > 1 || isOnGlobalTeam) {
        return (
          <TeamsDropdown
            currentUserTeams={userTeams}
            selectedTeamId={currentTeamId}
            onChange={handleTeamChange}
            includeAll={false}
            includeNoTeams
          />
        );
      }
      if (!isOnGlobalTeam && userTeams.length === 1) {
        return <h1>{userTeams[0].name}</h1>;
      }
    }
    return <h1>Controls</h1>;
  };

  const renderHeader = () => (
    <div className={`${baseClass}__header`}>
      <div className={`${baseClass}__text`}>
        <div className={`${baseClass}__title`}>{renderHeaderContent()}</div>
      </div>
    </div>
  );

  return (
    <MainContent>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header-wrap`}>{renderHeader()}</div>
        </div>
        {renderBody()}
      </div>
    </MainContent>
  );
};

export default ManageControlsPage;
