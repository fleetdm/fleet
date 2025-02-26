import React, { useContext } from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import TabNav from "components/TabNav";
import MainContent from "components/MainContent";
import TabText from "components/TabText";
import classnames from "classnames";

interface ISettingSubNavItem {
  name: string;
  pathname: string;
  exclude?: boolean;
}

interface ISettingsWrapperProp {
  children: JSX.Element;
  location: {
    pathname: string;
  };
  router: InjectedRouter; // v3
}

const baseClass = "admin-wrapper";

const AdminWrapper = ({
  children,
  location: { pathname },
  router,
}: ISettingsWrapperProp): JSX.Element => {
  const { isPremiumTier, isSandboxMode } = useContext(AppContext);

  const settingsSubNav: ISettingSubNavItem[] = [
    {
      name: "Organization settings",
      pathname: PATHS.ADMIN_ORGANIZATION,
      exclude: isSandboxMode,
    },
    {
      name: "Integrations",
      pathname: PATHS.ADMIN_INTEGRATIONS,
    },
    {
      name: "Users",
      pathname: PATHS.ADMIN_USERS,
      exclude: isSandboxMode,
    },
    {
      name: "Teams",
      pathname: PATHS.ADMIN_TEAMS,
      exclude: !isPremiumTier,
    },
  ];

  const filteredSettingsSubNav = settingsSubNav.filter((navItem) => {
    return !navItem.exclude;
  });

  const navigateToNav = (i: number): void => {
    const navPath = filteredSettingsSubNav[i].pathname;
    router.push(navPath);
  };

  const getTabIndex = (path: string): number => {
    return filteredSettingsSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that start with same pathname
      return path.startsWith(navItem.pathname);
    });
  };

  // we add a conditional sandbox-mode class here as we will need to make some
  // styling changes on the settings page to have the sticky elements work
  // with the sandbox mode expiry message
  const classNames = classnames(baseClass, { "sandbox-mode": isSandboxMode });

  return (
    <MainContent className={classNames}>
      <div className={`${baseClass}_wrapper`}>
        <TabNav>
          <h1>Settings</h1>
          <Tabs
            selectedIndex={getTabIndex(pathname)}
            onSelect={(i) => navigateToNav(i)}
          >
            <TabList>
              {filteredSettingsSubNav.map((navItem) => {
                // Bolding text when the tab is active causes a layout shift
                // so we add a hidden pseudo element with the same text string
                return (
                  <Tab key={navItem.name} data-text={navItem.name}>
                    <TabText>{navItem.name}</TabText>
                  </Tab>
                );
              })}
            </TabList>
          </Tabs>
        </TabNav>
        {children}
      </div>
    </MainContent>
  );
};

export default AdminWrapper;
