import React from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import generateTableHeaders from "./SoftwareTableConfig";
import EmptySoftware from "../../../software/components/EmptySoftware";

interface ISoftwareCardProps {
  errorSoftware: Error | null;
  isCollectingInventory: boolean;
  isSoftwareFetching: boolean;
  isSoftwareEnabled?: boolean;
  software: any;
  pageIndex: number;
  navTabIndex: any;
  onTabChange: any;
  onQueryChange: any;
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
}: ISoftwareCardProps): JSX.Element => {
  const tableHeaders = generateTableHeaders();

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
                  columns={tableHeaders}
                  data={(isSoftwareEnabled && software?.software) || []}
                  isLoading={isSoftwareFetching}
                  defaultSortHeader={"hosts_count"}
                  defaultSortDirection={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"software"}
                  emptyComponent={() =>
                    EmptySoftware(
                      (!isSoftwareEnabled && "disabled") ||
                        (isCollectingInventory && "collecting") ||
                        "default"
                    )
                  }
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  pageSize={SOFTWARE_DEFAULT_PAGE_SIZE}
                  onQueryChange={onQueryChange}
                />
              )}
            </TabPanel>
            <TabPanel>
              {!isSoftwareFetching && errorSoftware ? (
                <TableDataError />
              ) : (
                <TableContainer
                  columns={tableHeaders}
                  data={(isSoftwareEnabled && software?.software) || []}
                  isLoading={isSoftwareFetching}
                  defaultSortHeader={SOFTWARE_DEFAULT_SORT_HEADER}
                  defaultSortDirection={SOFTWARE_DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"software"}
                  emptyComponent={() =>
                    EmptySoftware(
                      (!isSoftwareEnabled && "disabled") ||
                        (isCollectingInventory && "collecting") ||
                        "default"
                    )
                  }
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  pageSize={SOFTWARE_DEFAULT_PAGE_SIZE}
                  onQueryChange={onQueryChange}
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
