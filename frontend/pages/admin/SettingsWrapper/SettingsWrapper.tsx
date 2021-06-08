import React from "react";
import { Tab, Tabs, TabList } from "react-tabs";
import { push } from "react-router-redux";
import { useDispatch, useSelector } from "react-redux";
import { IConfig } from "interfaces/config";
import permissionUtils from "utilities/permissions";

import PATHS from "router/paths";

interface ISettingSubNavItem {
  name: string;
  pathname: string;
}

interface IRootState {
  app: {
    config: IConfig;
  };
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
}

const getTabIndex = (path: string): number => {
  return settingsSubNav.findIndex((navItem) => {
    return navItem.pathname.includes(path);
  });
};

const baseClass = "settings-wrapper";

const SettingsWrapper = (props: ISettingsWrapperProp): JSX.Element => {
  const {
    children,
    location: { pathname },
  } = props;

  // Add Teams tab for basic tier only
  const config = useSelector((state: IRootState) => state.app.config);

  if (settingsSubNav.length === 2 && permissionUtils.isBasicTier(config)) {
    settingsSubNav.push({
      name: "Teams",
      pathname: PATHS.ADMIN_TEAMS,
    });
  }

  const dispatch = useDispatch();

  const navigateToNav = (i: number): void => {
    const navPath = settingsSubNav[i].pathname;
    dispatch(push(navPath));
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__nav-header`}>
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
      </div>
      {children}
    </div>
  );
};

export default SettingsWrapper;
