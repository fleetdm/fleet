import React, { useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import paths from "router/paths";
import softwareAPI, { ISoftwareResponse } from "services/entities/software";

import TabsWrapper from "components/TabsWrapper";
import TableContainer, { ITableQueryData } from "components/TableContainer";
import TableDataError from "components/TableDataError"; // TODO how do we handle errors? UI just keeps spinning?
// @ts-ignore
import Spinner from "components/Spinner";
import renderLastUpdatedText from "components/LastUpdatedText/LastUpdatedText";
import generateTableHeaders from "./SoftwareTableConfig";

interface ISoftwareCardProps {
  currentTeamId?: number;
  showSoftwareUI: boolean;
  setShowSoftwareUI: (showSoftwareTitle: boolean) => void;
  setActionURL?: (url: string) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "home-software";

const EmptySoftware = (message: string): JSX.Element => {
  return (
    <div className={`${baseClass}__empty-software`}>
      <h1>
        No installed software{" "}
        {message === "vulnerable"
          ? "with detected vulnerabilities"
          : "detected"}
        .
      </h1>
      <p>
        Expecting to see software? Check out the Fleet documentation on{" "}
        <a
          href="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
          target="_blank"
          rel="noopener noreferrer"
        >
          how to configure software inventory
        </a>
        .
      </p>
    </div>
  );
};

const Software = ({
  currentTeamId,
  showSoftwareUI,
  setShowSoftwareUI,
  setActionURL,
  setTitleDetail,
}: ISoftwareCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState<number>(0);
  const [pageIndex, setPageIndex] = useState<number>(0);

  const {
    data: software,
    isFetching: isSoftwareFetching,
    error: errorSoftware,
  } = useQuery<ISoftwareResponse, Error>(
    [
      "software",
      {
        pageIndex,
        pageSize: PAGE_SIZE,
        sortDirection: DEFAULT_SORT_DIRECTION,
        sortHeader: DEFAULT_SORT_HEADER,
        teamId: currentTeamId,
        vulnerable: !!navTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
      },
    ],
    () =>
      softwareAPI.load({
        page: pageIndex,
        perPage: PAGE_SIZE,
        orderKey: DEFAULT_SORT_HEADER,
        orderDir: DEFAULT_SORT_DIRECTION,
        vulnerable: !!navTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
        teamId: currentTeamId,
      }),
    {
      keepPreviousData: true,
      staleTime: 30000, // TODO: Discuss a reasonable staleTime given that counts are only updated infrequently?
      onSuccess: (data) => {
        setShowSoftwareUI(true);
        setTitleDetail &&
          setTitleDetail(
            renderLastUpdatedText(data.counts_updated_at, "software")
          );
      },
    }
  );

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onQueryChange = async ({
    pageIndex: newPageIndex,
  }: ITableQueryData) => {
    if (pageIndex !== newPageIndex) {
      setPageIndex(newPageIndex);
    }
  };

  const onTabChange = (index: number) => {
    const { MANAGE_SOFTWARE } = paths;
    setNavTabIndex(index);
    setActionURL &&
      setActionURL(
        index === 1 ? `${MANAGE_SOFTWARE}?vulnerable=true` : MANAGE_SOFTWARE
      );
  };

  const tableHeaders = generateTableHeaders();

  // Renders opaque information as host information is loading
  const opacity = showSoftwareUI ? { opacity: 1 } : { opacity: 0 };

  return (
    <div className={baseClass}>
      {!showSoftwareUI && (
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
                  data={software?.software || []}
                  isLoading={isSoftwareFetching}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"software"}
                  emptyComponent={EmptySoftware}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  pageSize={PAGE_SIZE}
                  onQueryChange={onQueryChange}
                />
              )}
            </TabPanel>
            <TabPanel>
              <TableContainer
                columns={tableHeaders}
                data={software?.software || []}
                isLoading={isSoftwareFetching}
                defaultSortHeader={DEFAULT_SORT_HEADER}
                defaultSortDirection={DEFAULT_SORT_DIRECTION}
                hideActionButton
                resultsTitle={"software"}
                emptyComponent={() => EmptySoftware("vulnerable")}
                showMarkAllPages={false}
                isAllPagesSelected={false}
                disableCount
                disableActionButton
                pageSize={PAGE_SIZE}
                onQueryChange={onQueryChange}
              />
            </TabPanel>
          </Tabs>
        </TabsWrapper>
      </div>
    </div>
  );
};

export default Software;
