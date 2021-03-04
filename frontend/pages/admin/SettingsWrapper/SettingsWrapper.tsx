import React from 'react';
import { Tab, Tabs, TabList } from 'react-tabs';
import { push } from 'react-router-redux';
import { useDispatch } from 'react-redux';

import PATHS from 'router/paths';

interface ISettingSubNavItem {
  name: string;
  pathname: string;
}

const settingsSubNav: ISettingSubNavItem[] = [
  {
    name: 'Organization Settings',
    pathname: PATHS.ADMIN_SETTINGS,
  },
  {
    name: 'Users',
    pathname: PATHS.ADMIN_USERS,
  },
  {
    name: 'Osquery Options',
    pathname: PATHS.ADMIN_OSQUERY,
  },
];

interface ISettingsWrapperProp {
  children: JSX.Element,
  location: {
    pathname: string
  }
}

const getDefaultTabIndex = (path: string): number => {
  return settingsSubNav.findIndex((navItem) => {
    return navItem.pathname.includes(path);
  });
};

const SettingsWrapper = (props: ISettingsWrapperProp): JSX.Element => {
  const { children, location: { pathname } } = props;
  const dispatch = useDispatch();

  const navigateToNav = (i: number): void => {
    const navPath = settingsSubNav[i].pathname;
    dispatch(push(navPath));
  };

  return (
    <div className="settings-wrapper">
      <h1>Settings</h1>
      <Tabs defaultIndex={getDefaultTabIndex(pathname)} onSelect={i => navigateToNav(i)}>
        <TabList>
          {settingsSubNav.map((navItem) => {
            return <Tab>{navItem.name}</Tab>;
          })}
        </TabList>
      </Tabs>
      {children}
    </div>
  );
};

export default SettingsWrapper;
