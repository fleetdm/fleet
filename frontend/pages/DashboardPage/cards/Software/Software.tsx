import React, { useContext, useMemo } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { Row } from "react-table";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import { buildQueryStringFromParams } from "utilities/url";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";

import generateTableHeaders from "./SoftwareTableConfig";

interface ISoftwareCardProps {
  errorSoftware: Error | null;
  isCollectingInventory: boolean;
  isSoftwareFetching: boolean;
  isSoftwareEnabled?: boolean;
  software: any;
  teamId?: number;
  pageIndex: number;
  navTabIndex: any;
  onTabChange: any;
  onQueryChange: any;
  router: InjectedRouter;
}

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

const SOFTWARE_DEFAULT_SORT_DIRECTION = "desc";
const SOFTWARE_DEFAULT_SORT_HEADER = "hosts_count";
const SOFTWARE_DEFAULT_PAGE_SIZE = 8;

const baseClass = "home-software";

const Software = ({
  errorSoftware,
  isCollectingInventory,
  isSoftwareFetching,
  isSoftwareEnabled,
  navTabIndex,
  onTabChange,
  onQueryChange,
  software,
  teamId,
  router,
}: ISoftwareCardProps): JSX.Element => {
  const { noSandboxHosts } = useContext(AppContext);

  const tableHeaders = useMemo(() => generateTableHeaders(teamId), [teamId]);

  const handleRowSelect = (row: IRowProps) => {
    const queryParams = { software_id: row.original.id, team_id: teamId };

    const path = queryParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(queryParams)}`
      : PATHS.MANAGE_HOSTS;

    router.push(path);
  };

  // Renders opaque information as host information is loading
  const opacity = isSoftwareFetching ? { opacity: 0 } : { opacity: 1 };

  return (
    <div className={baseClass}>
      {isSoftwareFetching && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        <TabsWrapper>
          <Tabs selectedIndex={navTabIndex} onSelect={onTabChange}>
            <TabList>
              <Tab>All</Tab>
              <Tab>Vulnerable</Tab>
            </TabList>
            <TabPanel>
              {!isSoftwareFetching && errorSoftware ? (
                <TableDataError />
              ) : (
                <TableContainer
                  columnConfigs={tableHeaders}
                  data={(isSoftwareEnabled && software?.software) || []}
                  isLoading={isSoftwareFetching}
                  defaultSortHeader={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  defaultSortDirection={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  resultsTitle="software"
                  emptyComponent={() => (
                    <EmptySoftwareTable
                      isCollectingSoftware={isCollectingInventory}
                    />
                  )}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  pageSize={SOFTWARE_DEFAULT_PAGE_SIZE}
                  onQueryChange={onQueryChange}
                  disableMultiRowSelect
                  onSelectSingleRow={handleRowSelect}
                />
              )}
            </TabPanel>
            <TabPanel>
              {!isSoftwareFetching && errorSoftware ? (
                <TableDataError />
              ) : (
                <TableContainer
                  columnConfigs={tableHeaders}
                  data={(isSoftwareEnabled && software?.software) || []}
                  isLoading={isSoftwareFetching}
                  defaultSortHeader={SOFTWARE_DEFAULT_SORT_HEADER}
                  defaultSortDirection={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  resultsTitle="software"
                  emptyComponent={() => (
                    <EmptySoftwareTable
                      isCollectingSoftware={isCollectingInventory}
                      softwareFilter="vulnerableSoftware"
                    />
                  )}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  pageSize={SOFTWARE_DEFAULT_PAGE_SIZE}
                  onQueryChange={onQueryChange}
                  disableMultiRowSelect
                  onSelectSingleRow={handleRowSelect}
                />
              )}
            </TabPanel>
          </Tabs>
        </TabsWrapper>
      </div>
    </div>
  );
};

export default Software;
