/**
software/library Library tab > Table
*/

import React, { useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";
import { getNextLocationPath } from "utilities/helpers";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import { ISoftwareTitlesResponse } from "services/entities/software";
import { ISoftwareTitle } from "interfaces/software";

import TableContainer from "components/TableContainer";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import LastUpdatedText from "components/LastUpdatedText";
import Slider from "components/forms/fields/Slider";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";

import generateLibraryTableConfig from "./SoftwareLibraryTableConfig";

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

interface ISoftwareLibraryTableProps {
  router: InjectedRouter;
  data?: ISoftwareTitlesResponse;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  selfServiceOnly: boolean;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
}

const baseClass = "software-library-table";

const SoftwareLibraryTable = ({
  router,
  data,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  selfServiceOnly,
  currentPage,
  teamId,
  isLoading,
}: ISoftwareLibraryTableProps) => {
  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "searchQuery":
            return val !== query;
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
    [currentPage, orderDirection, orderKey, query]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      const newQueryParam: Record<string, string | number | undefined> = {
        query: newTableQuery.searchQuery,
        fleet_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
      if (selfServiceOnly) {
        newQueryParam.self_service = "true";
      }

      return newQueryParam;
    },
    [selfServiceOnly, teamId]
  );

  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedParam = determineQueryParamChange(newTableQuery);

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_LIBRARY,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  const tableData: ISoftwareTitle[] | undefined = data?.software_titles;

  const softwareTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateLibraryTableConfig(router, teamId);
  }, [data, router, teamId]);

  // Determines if a user should be able to filter or search in the table
  const hasData = tableData && tableData.length > 0;
  const hasQuery = query !== "";

  const showFilterHeaders =
    isSoftwareEnabled && (hasData || hasQuery || selfServiceOnly);

  const handleSelfServiceToggle = () => {
    const queryParams: Record<string, string | number | undefined> = {
      query,
      fleet_id: teamId,
      order_direction: orderDirection,
      order_key: orderKey,
      page: 0,
    };
    if (!selfServiceOnly) {
      queryParams.self_service = "true";
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_LIBRARY,
        routeTemplate: "",
        queryParams,
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    if (!row.original.id) return;

    const detailsPath = PATHS.SOFTWARE_TITLE_DETAILS(
      row.original.id.toString()
    );

    router.push(getPathWithQueryParams(detailsPath, { fleet_id: teamId }));
  };

  const renderSoftwareCount = () => {
    return (
      <>
        <TableCount name="items" count={data?.count} />
        {tableData && data?.counts_updated_at && (
          <LastUpdatedText
            lastUpdatedAt={data.counts_updated_at}
            customTooltipText={
              <>
                The last time software data was <br />
                updated, including vulnerabilities <br />
                and host counts.
              </>
            }
          />
        )}
      </>
    );
  };

  const renderCustomControls = () => {
    return (
      <Slider
        value={selfServiceOnly}
        onChange={handleSelfServiceToggle}
        inactiveText="Only self-service"
        activeText="Only self-service"
      />
    );
  };

  const renderTableHelpText = () => (
    <div>
      Seeing unexpected software?{" "}
      <CustomLink
        url={GITHUB_NEW_ISSUE_LINK}
        text="File an issue on GitHub"
        newTab
      />
    </div>
  );

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={softwareTableHeaders}
        data={tableData ?? []}
        isLoading={isLoading}
        resultsTitle="items"
        emptyComponent={() => {
          if (!isSoftwareEnabled) {
            return (
              <EmptySoftwareTable isSoftwareDisabled />
            );
          }
          if (query !== "" || selfServiceOnly) {
            return (
              <EmptyState
                header="No items match the current search criteria"
                info="Expecting to see software? Check back later."
              />
            );
          }
          return (
            <EmptyState
              header="No software available"
              info="Add software to your library to get started."
              primaryButton={
                <Button
                  onClick={() =>
                    router.push(
                      getPathWithQueryParams(
                        PATHS.SOFTWARE_ADD_FLEET_MAINTAINED,
                        { fleet_id: teamId }
                      )
                    )
                  }
                >
                  Add software
                </Button>
              }
            />
          );
        }}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        pageIndex={currentPage}
        defaultSearchQuery={query}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableNextPage={!data?.meta.has_next_results}
        searchable={showFilterHeaders}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        additionalQueries={String(selfServiceOnly)}
        customControl={showFilterHeaders ? renderCustomControls : undefined}
        stackControls
        renderCount={renderSoftwareCount}
        renderTableHelpText={renderTableHelpText}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareLibraryTable;
