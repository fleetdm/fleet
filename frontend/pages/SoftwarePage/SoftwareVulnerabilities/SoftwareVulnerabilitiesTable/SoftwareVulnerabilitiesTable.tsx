import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import { IVulnerabilitiesResponse } from "services/entities/vulnerabilities";
import { buildQueryStringFromParams } from "utilities/url";
import { getNextLocationPath } from "utilities/helpers";

import generateTableConfig from "./VulnerabilitiesTableConfig";

const baseClass = "software-vulnerabilities-table";

interface IRowProps extends Row {
  original: {
    cve?: string;
  };
}

interface ISoftwareVulnerabilitiesTableProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  data?: IVulnerabilitiesResponse;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
}

const SoftwareVulnerabilitiesTable = ({
  router,
  isSoftwareEnabled,
  data,
  perPage,
  orderDirection,
  orderKey,
  currentPage,
  teamId,
  isLoading,
}: ISoftwareVulnerabilitiesTableProps) => {
  const { isSandboxMode, noSandboxHosts } = useContext(AppContext);

  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "sortDirection":
            return val !== orderDirection;
          case "sortHeader":
            return val !== orderKey;
          case "pageIndex":
            return val !== currentPage;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [currentPage, orderDirection, orderKey]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      return {
        team_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
    },
    [teamId]
  );

  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_OS,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  const softwareTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(teamId, router, {
      includeName: true,
      includeVulnerabilities: true,
      includeIcon: true,
    });
  }, [data, router, teamId]);

  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      cve: row.original.cve,
      team_id: teamId,
    };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsBySoftwareParams
    )}`;

    router.push(path);
  };

  const getItemsCountText = () => {
    const count = data?.count;
    if (!data?.vulnerabilities || !count) return "";

    return count === 1 ? `${count} item` : `${count} items`;
  };

  const getLastUpdatedText = () => {
    if (!data?.vulnerabilities || !data?.counts_updated_at) return "";
    return (
      <LastUpdatedText
        lastUpdatedAt={data.counts_updated_at}
        whatToRetrieve={"software"}
      />
    );
  };

  const renderSoftwareCount = () => {
    const itemText = getItemsCountText();
    const lastUpdatedText = getLastUpdatedText();

    if (!itemText) return null;

    return (
      <div className={`${baseClass}__count`}>
        <span>{itemText}</span>
        {lastUpdatedText}
      </div>
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

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={softwareTableHeaders}
        data={data?.vulnerabilities ?? []}
        isLoading={isLoading}
        resultsTitle={"items"}
        emptyComponent={() => (
          <EmptySoftwareTable
            isSoftwareDisabled={!isSoftwareEnabled}
            isSandboxMode={isSandboxMode}
            noSandboxHosts={noSandboxHosts}
          />
        )}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        defaultPageIndex={currentPage}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableNextPage={!data?.meta.has_next_results}
        searchable={false}
        onQueryChange={onQueryChange}
        stackControls
        renderCount={renderSoftwareCount}
        renderFooter={renderTableFooter}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
