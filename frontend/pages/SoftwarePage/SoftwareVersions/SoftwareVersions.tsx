import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { Row } from "react-table";

import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareQueryKey,
  ISoftwareVersionsResponse,
} from "services/entities/software";
import { AppContext } from "context/app";
import {
  GITHUB_NEW_ISSUE_LINK,
  VULNERABLE_DROPDOWN_OPTIONS,
} from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableDataError from "components/DataError";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";

import EmptySoftwareTable from "../components/EmptySoftwareTable";

import generateSoftwareVersionsTableHeaders from "./SoftwareVersionsTableConfig";

const baseClass = "software-versions";

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

interface ISoftwareVersionsProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  showVulnerableSoftware: boolean;
  currentPage: number;
  teamId?: number;
}

const SoftwareVersions = ({
  router,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  showVulnerableSoftware,
  currentPage,
  teamId,
}: ISoftwareVersionsProps) => {
  const { isSandboxMode, noSandboxHosts, isPremiumTier } = useContext(
    AppContext
  );

  // request to get software versions data
  const {
    data: softwareVersionsData,
    isLoading: isSoftwareVersionsLoading,
    isError: isSoftwareVersionsError,
  } = useQuery<
    ISoftwareVersionsResponse,
    Error,
    ISoftwareVersionsResponse,
    ISoftwareQueryKey[]
  >(
    [
      {
        scope: "software",
        page: currentPage,
        perPage,
        query,
        orderDirection,
        orderKey,
        teamId,
        vulnerable: showVulnerableSoftware,
      },
    ],
    ({ queryKey }) => softwareAPI.getSoftwareVersions(queryKey[0]),
    {
      keepPreviousData: true,
      // stale time can be adjusted if fresher data is desired based on
      // software inventory interval
      staleTime: 30000,
    }
  );

  // determines if a user be able to search in the table
  const searchable =
    isSoftwareEnabled &&
    (!!softwareVersionsData?.software ||
      query !== "" ||
      showVulnerableSoftware);

  const softwareTableHeaders = useMemo(
    () =>
      generateSoftwareVersionsTableHeaders(
        router,
        isPremiumTier,
        isSandboxMode,
        teamId
      ),
    [isPremiumTier, isSandboxMode, router, teamId]
  );

  const handleVulnFilterDropdownChange = (isFilterVulnerable: string) => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_TITLES,
        routeTemplate: "",
        queryParams: {
          query,
          teamId,
          orderDirection,
          orderKey,
          vulnerable: isFilterVulnerable,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    // const hostsBySoftwareParams = {
    //   software_id: row.original.id,
    //   team_id: teamId,
    // };

    // const path = hostsBySoftwareParams
    //   ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
    //       hostsBySoftwareParams
    //     )}`
    //   : PATHS.MANAGE_HOSTS;

    // router.push(path);
    // TODO: navigation to software details page.
    console.log("selectedRow", row.id);
  };

  const onQueryChange = () => {
    console.log("onQueryChange");
  };

  const renderSoftwareCount = useCallback(() => {
    const lastUpdatedAt = softwareVersionsData?.counts_updated_at;

    if (!isSoftwareEnabled || !lastUpdatedAt) {
      return null;
    }

    if (isSoftwareVersionsError) {
      return (
        <span className={`${baseClass}__count count-error`}>
          Failed to load software count
        </span>
      );
    }

    const hostCount = softwareVersionsData?.count;
    if (hostCount) {
      return (
        <div
          className={`${baseClass}__count ${
            isSoftwareVersionsLoading ? "count-loading" : ""
          }`}
        >
          <span>{`${hostCount} software item${
            hostCount === 1 ? "" : "s"
          }`}</span>
          <LastUpdatedText
            lastUpdatedAt={lastUpdatedAt}
            whatToRetrieve={"software"}
          />
        </div>
      );
    }

    return null;
  }, [
    softwareVersionsData,
    isSoftwareVersionsLoading,
    isSoftwareVersionsError,
    isSoftwareEnabled,
  ]);

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={showVulnerableSoftware}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={handleVulnFilterDropdownChange}
        tableFilterDropdown
      />
    );
  };

  const renderTableFooter = () => {
    return (
      <div>
        Seeing unexpected software or vulnerabilities?{" "}
        <CustomLink
          url={GITHUB_NEW_ISSUE_LINK}
          text="File an issue on GitHub"
          newTab
        />
      </div>
    );
  };

  if (isSoftwareVersionsError) {
    return <TableDataError />;
  }

  return (
    <div className={baseClass}>
      <div className={baseClass}>
        <TableContainer
          columns={softwareTableHeaders}
          data={softwareVersionsData?.software || []}
          isLoading={isSoftwareVersionsLoading}
          resultsTitle={"items"}
          emptyComponent={() => (
            <EmptySoftwareTable
              isSoftwareDisabled={!isSoftwareEnabled}
              isFilterVulnerable={showVulnerableSoftware}
              isSandboxMode={isSandboxMode}
              isCollectingSoftware={false} // TODO: update with new API
              isSearching={query !== ""}
              noSandboxHosts={noSandboxHosts}
            />
          )}
          defaultSortHeader={orderKey}
          defaultSortDirection={orderDirection}
          defaultPageIndex={currentPage}
          defaultSearchQuery={query}
          manualSortBy
          pageSize={perPage}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableNextPage // TODO: update with new API
          searchable={searchable}
          inputPlaceHolder="Search by name or vulnerabilities (CVEs)"
          onQueryChange={onQueryChange}
          // additionalQueries serves as a trigger for the useDeepEffect hook
          // to fire onQueryChange for events happeing outside of
          // the TableContainer.
          additionalQueries={showVulnerableSoftware ? "vulnerable" : ""}
          customControl={searchable ? renderVulnFilterDropdown : undefined}
          stackControls
          renderCount={renderSoftwareCount}
          renderFooter={renderTableFooter}
          disableMultiRowSelect
          onSelectSingleRow={handleRowSelect}
        />
      </div>
    </div>
  );
};

export default SoftwareVersions;
