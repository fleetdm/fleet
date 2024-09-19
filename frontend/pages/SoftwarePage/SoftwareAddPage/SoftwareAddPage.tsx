import React, { useCallback } from "react";
import { Tab, TabList, Tabs } from "react-tabs";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";

import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import TabsWrapper from "components/TabsWrapper";

const baseClass = "software-add-page";

interface IAddSoftwareSubNavItem {
  name: string;
  pathname: string;
}

const addSoftwareSubNav: IAddSoftwareSubNavItem[] = [
  {
    name: "Fleet-maintained",
    pathname: PATHS.SOFTWARE_ADD_FLEET_MAINTAINED,
  },
  {
    name: "Package",
    pathname: PATHS.SOFTWARE_ADD_PACKAGE,
  },
  {
    name: "App store (VPP)",
    pathname: PATHS.SOFTWARE_ADD_APP_STORE,
  },
];

const getTabIndex = (path: string): number => {
  return addSoftwareSubNav.findIndex((navItem) => {
    // tab stays highlighted for paths that start with same pathname
    return path.startsWith(navItem.pathname);
  });
};

export interface ISoftwareAddPageQueryParams {
  team_id?: string;
  query?: string;
  page?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
}

interface ISoftwareAddPageProps {
  children: JSX.Element;
  location: Location<ISoftwareAddPageQueryParams>;
  router: InjectedRouter;
}

const SoftwareAddPage = ({
  children,
  location,
  router,
}: ISoftwareAddPageProps) => {
  const navigateToNav = useCallback(
    (i: number): void => {
      // Only query param to persist between tabs is team id
      const teamIdParam = buildQueryStringFromParams({
        team_id: location?.query.team_id,
      });

      const navPath = addSoftwareSubNav[i].pathname.concat(`?${teamIdParam}`);
      router.replace(navPath);
    },
    [location, router]
  );

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to software"
          path={PATHS.SOFTWARE_TITLES}
          className={`${baseClass}__back-to-software`}
        />
        <h1>Add Software</h1>
        <TabsWrapper>
          <Tabs
            selectedIndex={getTabIndex(location?.pathname || "")}
            onSelect={navigateToNav}
          >
            <TabList>
              {addSoftwareSubNav.map((navItem) => {
                return (
                  <Tab key={navItem.name} data-text={navItem.name}>
                    {navItem.name}
                  </Tab>
                );
              })}
            </TabList>
          </Tabs>
        </TabsWrapper>
        {React.cloneElement(children, {
          router,
          currentTeamId: location.query.team_id,
        })}
      </>
    </MainContent>
  );
};

export default SoftwareAddPage;
