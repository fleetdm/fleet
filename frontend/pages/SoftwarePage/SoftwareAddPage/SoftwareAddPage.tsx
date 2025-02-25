import React, { useCallback, useContext } from "react";
import { Tab, TabList, Tabs } from "react-tabs";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { QueryContext } from "context/query";
import useToggleSidePanel from "hooks/useToggleSidePanel";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import TabsWrapper from "components/TabsWrapper";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";

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
    name: "App Store (VPP)",
    pathname: PATHS.SOFTWARE_ADD_APP_STORE,
  },
  {
    name: "Custom package",
    pathname: PATHS.SOFTWARE_ADD_PACKAGE,
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
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(false);

  const navigateToNav = useCallback(
    (i: number): void => {
      setSidePanelOpen(false);
      // Only query param to persist between tabs is team id
      const teamIdParam = buildQueryStringFromParams({
        team_id: location.query.team_id,
      });

      const navPath = addSoftwareSubNav[i].pathname.concat(`?${teamIdParam}`);
      router.replace(navPath);
    },
    [location.query.team_id, router, setSidePanelOpen]
  );

  // Quick exit if no team_id param. This page must have a team id to function
  // correctly. We redirect to the same page with the "No team" context if it
  // is not provieded.
  if (!location.query.team_id) {
    router.replace(
      `${location.pathname}?${buildQueryStringFromParams({
        team_id: APP_CONTEXT_NO_TEAM_ID,
      })}`
    );
    return null;
  }

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const backUrl = `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
    team_id: location.query.team_id,
  })}`;

  return (
    <>
      <MainContent className={baseClass}>
        <>
          <BackLink
            text="Back to software"
            path={backUrl}
            className={`${baseClass}__back-to-software`}
          />
          <h1>Add software</h1>
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
            currentTeamId: parseInt(location.query.team_id, 10),
            isSidePanelOpen,
            setSidePanelOpen,
          })}
        </>
      </MainContent>
      {isSidePanelOpen && (
        <SidePanelContent>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={() => setSidePanelOpen(false)}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default SoftwareAddPage;
