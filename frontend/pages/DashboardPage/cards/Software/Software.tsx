import React, { useMemo } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { Row } from "react-table";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router";

import { getPathWithQueryParams } from "utilities/url";
import { ISoftwareResponse } from "interfaces/software";

import { ITableQueryData } from "components/TableContainer/TableContainer";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import TableContainer from "components/TableContainer";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";

import generateTableHeaders from "./SoftwareTableConfig";

interface ISoftwareCardProps {
  errorSoftware: Error | null;
  isSoftwareFetching: boolean;
  isSoftwareEnabled?: boolean;
  software?: ISoftwareResponse;
  teamId?: number;
  navTabIndex: number;
  onTabChange: (index: number, last: number, event: Event) => boolean | void;
  onQueryChange?:
    | ((queryData: ITableQueryData) => void)
    | ((queryData: ITableQueryData) => number);
  router: InjectedRouter;
  softwarePageIndex: number;
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
  isSoftwareFetching,
  isSoftwareEnabled,
  navTabIndex,
  onTabChange,
  onQueryChange,
  software,
  teamId,
  router,
  softwarePageIndex,
}: ISoftwareCardProps): JSX.Element => {
  const tableHeaders = useMemo(() => generateTableHeaders(teamId), [teamId]);

  const handleRowSelect = (row: IRowProps) => {
    const path = getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
      software_id: row.original.id,
      team_id: teamId,
    });

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
        <TabNav>
          <Tabs selectedIndex={navTabIndex} onSelect={onTabChange}>
            <TabList>
              <Tab>
                <TabText>All</TabText>
              </Tab>
              <Tab>
                <TabText>Vulnerable</TabText>
              </Tab>
            </TabList>
            <TabPanel>
              {!isSoftwareFetching && errorSoftware ? (
                <DataError verticalPaddingSize="pad-large" />
              ) : (
                <TableContainer
                  columnConfigs={tableHeaders}
                  data={(isSoftwareEnabled && software?.software) || []}
                  isLoading={isSoftwareFetching}
                  pageIndex={softwarePageIndex}
                  defaultSortHeader={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  defaultSortDirection={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  resultsTitle="software"
                  emptyComponent={() => <EmptySoftwareTable />}
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
                <DataError verticalPaddingSize="pad-large" />
              ) : (
                <TableContainer
                  columnConfigs={tableHeaders}
                  data={(isSoftwareEnabled && software?.software) || []}
                  isLoading={isSoftwareFetching}
                  pageIndex={softwarePageIndex}
                  defaultSortHeader={SOFTWARE_DEFAULT_SORT_HEADER}
                  defaultSortDirection={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  resultsTitle="software"
                  emptyComponent={() => (
                    <EmptySoftwareTable vulnFilters={{ vulnerable: true }} />
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
        </TabNav>
      </div>
    </div>
  );
};

export default Software;
