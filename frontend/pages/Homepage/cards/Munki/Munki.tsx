import React, { useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import macadminsAPI from "services/entities/macadmins";
import {
  IMacadminAggregate,
  IMunkiIssuesAggregate,
  IMunkiVersionsAggregate,
} from "interfaces/macadmins";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import LastUpdatedText from "components/LastUpdatedText";
import generateMunkiVersionsTableHeaders from "./MunkiVersionsTableConfig";
import generateMunkiIssuesTableHeaders from "./MunkiIssuesTableConfig";

interface IMunkiCardProps {
  showMunkiUI: boolean;
  currentTeamId: number | undefined;
  setShowMunkiUI: (showMunkiTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "home-munki";

const EmptyMunkiIssues = (): JSX.Element => (
  <div className={`${baseClass}__empty-munki`}>
    <h1>No Munki issues detected</h1>
    <p>
      This report is updated every hour to protect the performance of your
      devices.
    </p>
  </div>
);

const EmptyMunkiVersions = (): JSX.Element => (
  <div className={`${baseClass}__empty-munki`}>
    <h1>Unable to detect Munki versions</h1>
    <p>
      To see Munki versions, deploy&nbsp;
      <a
        href="https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer"
        target="_blank"
        rel="noopener noreferrer"
      >
        Fleet&apos;s osquery installer
      </a>
      .
    </p>
  </div>
);

const Munki = ({
  showMunkiUI,
  currentTeamId,
  setShowMunkiUI,
  setTitleDetail,
}: IMunkiCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState<number>(0);
  const [pageIndex, setPageIndex] = useState<number>(0);
  const [munkiIssuesData, setMunkiIssuesData] = useState<
    IMunkiIssuesAggregate[]
  >([]);
  const [munkiVersionsData, setMunkiVersionsData] = useState<
    IMunkiVersionsAggregate[]
  >([]);

  const { isFetching: isMunkiFetching, error: errorMunki } = useQuery<
    IMacadminAggregate,
    Error
  >(["munki", currentTeamId], () => macadminsAPI.loadAll(currentTeamId), {
    keepPreviousData: true,
    onSuccess: (data) => {
      const {
        counts_updated_at,
        munki_versions,
        munki_issues,
      } = data.macadmins;

      setMunkiVersionsData(munki_versions);
      setMunkiIssuesData(munki_issues);
      setShowMunkiUI(true);
      setTitleDetail &&
        setTitleDetail(
          <LastUpdatedText
            lastUpdatedAt={counts_updated_at}
            whatToRetrieve={"Munki versions"}
          />
        );
    },
    onError: () => {
      setShowMunkiUI(true);
    },
  });

  const onTabChange = (index: number) => {
    setNavTabIndex(index);
  };

  const munkiVersionsTableHeaders = generateMunkiVersionsTableHeaders();
  const munkiIssuesTableHeaders = generateMunkiIssuesTableHeaders();

  // Renders opaque information as host information is loading
  const opacity = showMunkiUI ? { opacity: 1 } : { opacity: 0 };

  return (
    <div className={baseClass}>
      {!showMunkiUI && (
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
              {errorMunki ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={munkiIssuesTableHeaders}
                  data={munkiIssuesData || []}
                  isLoading={isMunkiFetching}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"Munki"}
                  emptyComponent={EmptyMunkiIssues}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  disablePagination
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
            <TabPanel>
              {errorMunki ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={munkiVersionsTableHeaders}
                  data={munkiVersionsData || []}
                  isLoading={isMunkiFetching}
                  defaultSortHeader={DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"Munki"}
                  emptyComponent={EmptyMunkiVersions}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
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
