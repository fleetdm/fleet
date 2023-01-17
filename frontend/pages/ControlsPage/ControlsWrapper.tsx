import React, { useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";

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
  location: {
    pathname: string;
  };
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
  location: { pathname },
  router,
}: IControlsWrapperProp): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const navigateToNav = (i: number): void => {
    const navPath = controlsSubNav[i].pathname;
    router.push(navPath);
  };

  if (isPremiumTier) {
    return (
      <MainContent className={baseClass}>
        <div className={`${baseClass}_wrapper}`}>
          <TabsWrapper>
            {/* TODO: replace below heading with teams dropdown - defaults to No Teams */}
            <h1>Controls</h1>
            <Tabs
              selectedIndex={getTabIndex(pathname)}
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
      </MainContent>
    );
  }
  return (
    // TODO - implement upsell empty state
    <h1>Buy Premium</h1>
  );
};

export default ControlsWrapper;
