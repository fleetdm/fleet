import React, { useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";
import TeamsDropdownHeader, {
  ITeamsDropdownState,
} from "components/PageHeader/TeamsDropdownHeader";

import { find } from "lodash";

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
  const { availableTeams, isPremiumTier, setCurrentTeam } = useContext(
    AppContext
  );

  const navigateToNav = (i: number): void => {
    const navPath = controlsSubNav[i].pathname;
    router.push(navPath);
  };

  const handleTeamSelect = (ctx: ITeamsDropdownState) => {
    const teamId = ctx.teamId;
    const queryString = teamId === undefined ? "" : `?team_id=${teamId}`;
    router.replace(location.pathname + queryString);
    const selectedTeam = find(availableTeams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const renderHeader = () => (
    <div className={`${baseClass}__header`}>
      <div className={`${baseClass}__text`}>
        <div className={`${baseClass}__title`}>
          <TeamsDropdownHeader
            router={router}
            location={location}
            baseClass={baseClass}
            defaultTitle="Controls"
            onChange={handleTeamSelect}
            description={() => {
              return null;
            }}
            includeNoTeams
            includeAll={false}
          />
        </div>
      </div>
    </div>
  );

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>{renderHeader()}</div>
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
      </div>
    </MainContent>
  );
};

export default ControlsWrapper;
