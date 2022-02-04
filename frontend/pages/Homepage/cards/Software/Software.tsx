import React, { useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import ReactTooltip from "react-tooltip";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import paths from "router/paths";
import softwareAPI, { ISoftwareResponse } from "services/entities/software";

import TabsWrapper from "components/TabsWrapper";
import TableContainer, { ITableQueryData } from "components/TableContainer";
import TableDataError from "components/TableDataError"; // TODO how do we handle errors? UI just keeps spinning?
// @ts-ignore
import Spinner from "components/Spinner";

import generateTableHeaders from "./SoftwareTableConfig";
import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

interface ISoftwareCardProps {
  currentTeamId?: number;
  setShowSoftwareUI: (showSoftwareTitle: boolean) => void;
  showSoftwareUI: boolean;
  setActionLink?: (url: string) => void;
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
          href="https://fleetdm.com/docs/deploying/configuration#software-inventory"
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

const renderLastUpdatedAt = (lastUpdatedAt: string) => {
  if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
    lastUpdatedAt = "never";
  } else {
    lastUpdatedAt = formatDistanceToNowStrict(new Date(lastUpdatedAt), {
      addSuffix: true,
    });
  }
  return (
    <span className="last-updated">
      {`Last updated ${lastUpdatedAt}`}
      <span className={`tooltip`}>
        <span
          className={`tooltip__tooltip-icon`}
          data-tip
          data-for="last-updated-tooltip"
          data-tip-disable={false}
        >
          <img alt="question icon" src={QuestionIcon} />
        </span>
        <ReactTooltip
          place="top"
          type="dark"
          effect="solid"
          backgroundColor="#3e4771"
          id="last-updated-tooltip"
          data-html
        >
          <span className={`tooltip__tooltip-text`}>
            Fleet periodically
            <br />
            queries all hosts
            <br />
            to retrieve software
          </span>
        </ReactTooltip>
      </span>
    </span>
  );
};

const Software = ({
  currentTeamId,
  setShowSoftwareUI,
  showSoftwareUI,
  setActionLink,
  setTitleDetail,
}: ISoftwareCardProps): JSX.Element => {
  const [isLoadingSoftware, setIsLoadingSoftware] = useState<boolean>(true);
  const [navTabIndex, setNavTabIndex] = useState<number>(0);
  const [pageIndex, setPageIndex] = useState<number>(0);

  const { data: software, error: errorSoftware } = useQuery<
    ISoftwareResponse,
    Error
  >(
    [
      "software",
      {
        pageIndex,
        pageSize: PAGE_SIZE,
        // searchQuery,
        sortDirection: DEFAULT_SORT_DIRECTION,
        sortHeader: DEFAULT_SORT_HEADER,
        teamId: currentTeamId,
        vulnerable: !!navTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
      },
    ],
    () => {
      setIsLoadingSoftware(true);
      return softwareAPI.load({
        page: pageIndex,
        perPage: PAGE_SIZE,
        // query: searchQuery,
        // TODO confirm sort is working?
        orderKey: DEFAULT_SORT_HEADER,
        orderDir: DEFAULT_SORT_DIRECTION,
        vulnerable: !!navTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
        teamId: currentTeamId,
      });
    },
    {
      // initialData: { software: [], counts_updated_at: "" },
      // placeholderData: { software: [], counts_updated_at: "" },
      // enabled: true,
      // If keepPreviousData is enabled,
      // useQuery no longer returns isLoading when making new calls after load
      // So we manage our own load states
      keepPreviousData: true,
      staleTime: 30000, // TODO: Discuss a reasonable staleTime given that counts are only updated infrequently?
      onSuccess: (data) => {
        setShowSoftwareUI(true);
        setIsLoadingSoftware(false);
        setTitleDetail &&
          setTitleDetail(renderLastUpdatedAt(data.counts_updated_at));
      },
      onError: () => {
        setIsLoadingSoftware(false);
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
    setActionLink &&
      setActionLink(
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
              {!isLoadingSoftware && errorSoftware ? (
                <TableDataError />
              ) : (
                <TableContainer
                  columns={tableHeaders}
                  data={software?.software || []}
                  isLoading={isLoadingSoftware}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  // manualSortBy
                  hideActionButton
                  resultsTitle={"software"}
                  emptyComponent={EmptySoftware}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  pageSize={PAGE_SIZE}
                  onQueryChange={onQueryChange}
                  // additionalQueries={navTabIndex ? "vulnerable" : ""}
                />
              )}
            </TabPanel>
            <TabPanel>
              <TableContainer
                columns={tableHeaders}
                data={software?.software || []}
                isLoading={isLoadingSoftware}
                defaultSortHeader={DEFAULT_SORT_HEADER}
                defaultSortDirection={DEFAULT_SORT_DIRECTION}
                // manualSortBy
                hideActionButton
                resultsTitle={"software"}
                emptyComponent={() => EmptySoftware("vulnerable")}
                showMarkAllPages={false}
                isAllPagesSelected={false}
                disableCount
                disableActionButton
                pageSize={PAGE_SIZE}
                onQueryChange={onQueryChange}
                // additionalQueries={navTabIndex ? "vulnerable" : ""}
              />
            </TabPanel>
          </Tabs>
        </TabsWrapper>
      </div>
    </div>
  );
};

export default Software;
