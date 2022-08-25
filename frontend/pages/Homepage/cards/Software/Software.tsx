import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { AppContext } from "context/app";
import paths from "router/paths";
import configAPI from "services/entities/config";
import softwareAPI, { ISoftwareResponse } from "services/entities/software";

import TabsWrapper from "components/TabsWrapper";
import TableContainer, { ITableQueryData } from "components/TableContainer";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import LastUpdatedText from "components/LastUpdatedText/LastUpdatedText";
import generateTableHeaders from "./SoftwareTableConfig";
import EmptySoftware from "../../../software/components/EmptySoftware";

interface ISoftwareCardProps {
  currentTeamId?: number;
  showSoftwareUI: boolean;
  setShowSoftwareUI: (showSoftwareTitle: boolean) => void;
  setActionURL?: (url: string) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_PAGE_SIZE = 8;
const baseClass = "home-software";

const Software = ({
  currentTeamId,
  showSoftwareUI,
  setShowSoftwareUI,
  setActionURL,
  setTitleDetail,
}: ISoftwareCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState<number>(0);
  const [pageIndex, setPageIndex] = useState<number>(0);
  const [isSoftwareEnabled, setIsSoftwareEnabled] = useState<boolean>();

  const { availableTeams, currentTeam, isOnGlobalTeam } = useContext(
    AppContext
  );

  const { data: config } = useQuery(["config"], configAPI.loadAll, {
    onSuccess: (data) => {
      setIsSoftwareEnabled(data?.features?.enable_software_inventory);
    },
  });

  const {
    data: software,
    isFetching: isSoftwareFetching,
    error: errorSoftware,
  } = useQuery<ISoftwareResponse, Error>(
    [
      "software",
      {
        pageIndex,
        pageSize: DEFAULT_PAGE_SIZE,
        sortDirection: DEFAULT_SORT_DIRECTION,
        sortHeader: DEFAULT_SORT_HEADER,
        teamId: currentTeamId,
        vulnerable: !!navTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
      },
    ],
    () =>
      softwareAPI.load({
        page: pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        orderKey: DEFAULT_SORT_HEADER,
        orderDir: DEFAULT_SORT_DIRECTION,
        vulnerable: !!navTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
        teamId: currentTeamId,
      }),
    {
      enabled:
        isOnGlobalTeam ||
        !!availableTeams?.find((t) => t.id === currentTeam?.id),
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
      onSuccess: (data) => {
        setShowSoftwareUI(true);
        if (isSoftwareEnabled && data.software?.length !== 0) {
          setTitleDetail &&
            setTitleDetail(
              <LastUpdatedText
                lastUpdatedAt={data.counts_updated_at}
                whatToRetrieve={"software"}
              />
            );
        }
      },
      onError: () => {
        setShowSoftwareUI(true);
      },
    }
  );

  // TODO: Rework after backend is adjusted to differentiate empty search/filter results from
  // collecting inventory
  const isCollectingInventory =
    !currentTeamId &&
    !pageIndex &&
    !software?.software &&
    software?.counts_updated_at === null;

  if (isCollectingInventory) {
    setTitleDetail && setTitleDetail("");
  }

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
                  pageSize={DEFAULT_PAGE_SIZE}
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
                  data={software?.software || []}
                  isLoading={isSoftwareFetching}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
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
                  pageSize={DEFAULT_PAGE_SIZE}
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
