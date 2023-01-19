import React, { useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import {
  IMunkiIssuesAggregate,
  IMunkiVersionsAggregate,
} from "interfaces/macadmins";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import EmptyTable from "components/EmptyTable";

import munkiVersionsTableHeaders from "./MunkiVersionsTableConfig";
import munkiIssuesTableHeaders from "./MunkiIssuesTableConfig";

interface IMunkiCardProps {
  errorMacAdmins: Error | null;
  isMacAdminsFetching: boolean;
  munkiIssuesData: IMunkiIssuesAggregate[];
  munkiVersionsData: IMunkiVersionsAggregate[];
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "home-munki";

const Munki = ({
  errorMacAdmins,
  isMacAdminsFetching,
  munkiIssuesData,
  munkiVersionsData,
}: IMunkiCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState<number>(0);

  const onTabChange = (index: number) => {
    setNavTabIndex(index);
  };

  // Renders opaque information as host information is loading
  const opacity = isMacAdminsFetching ? { opacity: 0 } : { opacity: 1 };

  return (
    <div className={baseClass}>
      {isMacAdminsFetching && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        <TabsWrapper>
          <Tabs selectedIndex={navTabIndex} onSelect={onTabChange}>
            <TabList>
              <Tab>Issues</Tab>
              <Tab>Versions</Tab>
            </TabList>
            <TabPanel>
              {errorMacAdmins ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={munkiIssuesTableHeaders}
                  data={munkiIssuesData || []}
                  isLoading={isMacAdminsFetching}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"Munki"}
                  emptyComponent={() => (
                    <EmptyTable
                      header="No Munki issues detected"
                      info="This report is updated every hour to protect the performance of your
      devices."
                    />
                  )}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  isClientSidePagination
                  disableCount
                  disableActionButton
                  disablePagination
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
            <TabPanel>
              {errorMacAdmins ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={munkiVersionsTableHeaders}
                  data={munkiVersionsData || []}
                  isLoading={isMacAdminsFetching}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"Munki"}
                  emptyComponent={() => (
                    <EmptyTable
                      header="Unable to detect Munki versions"
                      info={
                        <>
                          To see Munki versions, deploy&nbsp;
                          <a
                            href="https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer"
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            Fleet&apos;s osquery installer
                          </a>
                          .
                        </>
                      }
                    />
                  )}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  isClientSidePagination
                  disableCount
                  disableActionButton
                  disablePagination
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
          </Tabs>
        </TabsWrapper>
      </div>
    </div>
  );
};

export default Munki;
