import React, { useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import TabsWrapper from "components/TabsWrapper";

interface ISettingSubNavItem {
  name: string;
  pathname: string;
}

const settingsSubNav: ISettingSubNavItem[] = [
  {
    name: "Organization settings",
    pathname: PATHS.ADMIN_SETTINGS,
  },
  {
    name: "Users",
    pathname: PATHS.ADMIN_USERS,
  },
];

interface ISettingsWrapperProp {
  children: JSX.Element;
  location: {
    pathname: string;
  };
  router: InjectedRouter; // v3
}

const getTabIndex = (path: string): number => {
  return settingsSubNav.findIndex((navItem) => {
    return navItem.pathname.includes(path);
  });
};

const baseClass = "settings-wrapper";

const SettingsWrapper = ({
  children,
  location: { pathname },
  router,
}: ISettingsWrapperProp): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  
  if (isPremiumTier && settingsSubNav.length === 2) {
    settingsSubNav.push({
      name: "Teams",
      pathname: PATHS.ADMIN_TEAMS,
    });
  }

  const navigateToNav = (i: number): void => {
    const navPath = settingsSubNav[i].pathname;
    router.push(navPath);
  };

  return (
    <div className={`${baseClass} body-wrap`}>
      <TabsWrapper>
        <h1>Settings</h1>
        <Tabs
          selectedIndex={getTabIndex(pathname)}
          onSelect={(i) => navigateToNav(i)}
        >
          <TabList>
            {settingsSubNav.map((navItem) => {
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
      {children}
    </div>
  );
};

export default SettingsWrapper;
