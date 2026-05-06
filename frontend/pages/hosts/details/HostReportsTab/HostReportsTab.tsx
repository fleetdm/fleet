import React, { useState, useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { SingleValue } from "react-select-5";

import PATHS from "router/paths";
import hostReportsAPI, {
  IHostReport,
  IListHostReportsResponse,
} from "services/entities/host_reports";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { pluralize } from "utilities/strings/stringUtils";
import { getNextLocationPath } from "utilities/helpers";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SearchField from "components/forms/fields/SearchField";
import Slider from "components/forms/fields/Slider";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Pagination from "components/Pagination";
import EmptyState from "components/EmptyState";
import Button from "components/buttons/Button";

import HostReportCard from "./HostReportCard";
import EmptyReports from "./EmptyReports";

const baseClass = "host-reports-tab";

const PAGE_SIZE = 50;

type SortOption =
  | "newest_results"
  | "oldest_results"
  | "name_asc"
  | "name_desc";

const DEFAULT_SORT_OPTION: SortOption = "newest_results";

const SORT_OPTIONS: CustomOptionType[] = [
  { value: "newest_results", label: "Newest results" },
  { value: "oldest_results", label: "Oldest results" },
  { value: "name_asc", label: "Name A-Z" },
  { value: "name_desc", label: "Name Z-A" },
];

const getSortParams = (sort: SortOption) => {
  switch (sort) {
    case "newest_results":
      return { order_key: "last_fetched", order_direction: "desc" };
    case "oldest_results":
      return { order_key: "last_fetched", order_direction: "asc" };
    case "name_asc":
      return { order_key: "name", order_direction: "asc" };
    case "name_desc":
      return { order_key: "name", order_direction: "desc" };
    default:
      return { order_key: "last_fetched", order_direction: "desc" };
  }
};

interface IHostReportsTabProps {
  hostId: number;
  hostName: string;
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      query?: string;
      sort?: string;
      show_dont_store?: string;
    };
  };
  saveReportsDisabledInConfig?: boolean;
  showReportsEmptyState?: boolean;
  canScheduleReport?: boolean;
  onScheduleReport?: () => void;
}

const HostReportsTab = ({
  hostId,
  hostName,
  router,
  location,
  saveReportsDisabledInConfig,
  showReportsEmptyState = false,
  canScheduleReport,
  onScheduleReport,
}: IHostReportsTabProps): JSX.Element => {
  const searchQuery = location.query.query ?? "";
  const sortOption: SortOption =
    location.query.sort &&
    SORT_OPTIONS.some((o) => o.value === location.query.sort)
      ? (location.query.sort as SortOption)
      : DEFAULT_SORT_OPTION;
  const showDontStoreResults = location.query.show_dont_store === "true";
  const [page, setPage] = useState(0);

  const sortParams = getSortParams(sortOption);

  // If save reports is disabled in the org settings, always include reports
  // that don't store results and hide the toggle
  const includeReportsDontStoreResults =
    saveReportsDisabledInConfig || showDontStoreResults;

  const { data: reportsData, isLoading, isError, isFetching } = useQuery<
    IListHostReportsResponse,
    Error
  >(
    [
      "host_reports",
      hostId,
      page,
      sortParams.order_key,
      sortParams.order_direction,
      searchQuery,
      includeReportsDontStoreResults,
    ],
    () =>
      hostReportsAPI.list(hostId, {
        page,
        per_page: PAGE_SIZE,
        order_key: sortParams.order_key,
        order_direction: sortParams.order_direction,
        query: searchQuery || undefined,
        include_reports_dont_store_results: includeReportsDontStoreResults,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
    }
  );

  const reports = reportsData?.reports ?? [];
  const totalCount = reportsData?.count ?? 0;
  const meta = reportsData?.meta;

  const onSearchChange = useCallback(
    (value: string) => {
      router.replace(
        getNextLocationPath({
          pathPrefix: location.pathname,
          queryParams: { ...location.query, query: value || undefined },
        })
      );
      setPage(0);
    },
    [router, location.pathname, location.query]
  );

  const onSortChange = useCallback(
    (newValue: SingleValue<CustomOptionType>) => {
      if (!newValue) return;
      router.replace(
        getNextLocationPath({
          pathPrefix: location.pathname,
          queryParams: {
            ...location.query,
            sort:
              newValue.value === DEFAULT_SORT_OPTION
                ? undefined
                : newValue.value,
          },
        })
      );
      setPage(0);
    },
    [router, location.pathname, location.query]
  );

  const onToggleDontStoreResults = useCallback(() => {
    router.replace(
      getNextLocationPath({
        pathPrefix: location.pathname,
        queryParams: {
          ...location.query,
          show_dont_store: showDontStoreResults ? undefined : "true",
        },
      })
    );
    setPage(0);
  }, [router, location.pathname, location.query, showDontStoreResults]);

  const onShowDetails = useCallback(
    (report: IHostReport) => {
      router.push(PATHS.HOST_REPORT_RESULTS(hostId, report.report_id));
    },
    [hostId, router]
  );

  const onViewAllHosts = useCallback(
    (report: IHostReport) => {
      router.push(PATHS.REPORT_DETAILS(report.report_id));
    },
    [router]
  );

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    return <DataError />;
  }

  // No reports should be available if MDM enrollment is pending so hide any previous reports
  // that may be associated with the host to prevent confusion while pending
  if (showReportsEmptyState) {
    return <EmptyReports isSearching={false} />;
  }

  const isTrulyEmpty = totalCount === 0 && !searchQuery;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__controls`}>
        <div className={`${baseClass}__controls-left`}>
          <span className={`${baseClass}__count`}>
            {totalCount} {pluralize(totalCount, "report")}
          </span>
          {!saveReportsDisabledInConfig && (
            <Slider
              value={showDontStoreResults}
              onChange={onToggleDontStoreResults}
              activeText="Show reports that don't store results"
              inactiveText="Show reports that don't store results"
              className={`${baseClass}__toggle`}
              disabled={isTrulyEmpty}
            />
          )}
        </div>
        <div className={`${baseClass}__controls-right`}>
          <DropdownWrapper
            name="sort-reports"
            options={SORT_OPTIONS}
            value={sortOption}
            onChange={onSortChange}
            className={`${baseClass}__sort-dropdown`}
            variant="table-filter"
            isDisabled={isTrulyEmpty}
          />
          <SearchField
            placeholder="Search by name"
            defaultValue={searchQuery}
            onChange={onSearchChange}
            disabled={isTrulyEmpty}
          />
        </div>
      </div>

      {isTrulyEmpty ? (
        <EmptyState
          header="No reports scheduled"
          info="Select Refetch to load the latest data from this host, or schedule a report."
          primaryButton={
            canScheduleReport ? (
              <Button onClick={onScheduleReport} type="button">
                Schedule a report
              </Button>
            ) : undefined
          }
        />
      ) : reports.length === 0 && searchQuery ? (
        <EmptyReports isSearching />
      ) : (
        <>
          <div className={`${baseClass}__reports-list`}>
            {reports.map((report) => (
              <HostReportCard
                key={report.report_id}
                report={report}
                hostName={hostName}
                onShowDetails={onShowDetails}
                onViewAllHosts={onViewAllHosts}
              />
            ))}
          </div>
          <Pagination
            onNextPage={() => setPage((p) => p + 1)}
            onPrevPage={() => setPage((p) => Math.max(0, p - 1))}
            disableNext={!meta?.has_next_results}
            disablePrev={!meta?.has_previous_results}
            hidePagination={
              !!meta && !meta.has_next_results && !meta.has_previous_results
            }
          />
        </>
      )}
      {isFetching && !isLoading && (
        <div className={`${baseClass}__fetching-overlay`} />
      )}
    </div>
  );
};

export default HostReportsTab;
