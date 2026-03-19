import React, { useState, useCallback, useMemo } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import hostReportsAPI, {
  IHostReport,
  IListHostReportsResponse,
} from "services/entities/host_reports";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SearchField from "components/forms/fields/SearchField";
import Slider from "components/forms/fields/Slider";
import ActionsDropdown from "components/ActionsDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";
import Pagination from "components/Pagination";

import HostReportCard from "./HostReportCard";
import EmptyReports from "./EmptyReports";

const baseClass = "host-reports-tab";

const PAGE_SIZE = 50;

type SortOption =
  | "newest_results"
  | "oldest_results"
  | "name_asc"
  | "name_desc";

const SORT_OPTIONS: IDropdownOption[] = [
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
}

const HostReportsTab = ({
  hostId,
  hostName,
  router,
  location,
  saveReportsDisabledInConfig,
}: IHostReportsTabProps): JSX.Element => {
  const searchQuery = location.query.query ?? "";
  const sortOption: SortOption =
    location.query.sort &&
    ["newest_results", "oldest_results", "name_asc", "name_desc"].includes(
      location.query.sort
    )
      ? (location.query.sort as SortOption)
      : "newest_results";
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
    (value: string) => {
      router.replace(
        getNextLocationPath({
          pathPrefix: location.pathname,
          queryParams: {
            ...location.query,
            sort: value === "newest_results" ? undefined : value,
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
      router.push(PATHS.HOST_REPORT_RESULTS(hostId, report.query_id));
    },
    [hostId, router]
  );

  const onViewAllHosts = useCallback(
    (report: IHostReport) => {
      router.push(PATHS.REPORT_DETAILS(report.query_id));
    },
    [router]
  );

  const sortDropdownOptions = useMemo(() => {
    return SORT_OPTIONS.map((opt) => ({
      ...opt,
      label: opt.value === sortOption ? `Sort: ${opt.label}` : opt.label,
    }));
  }, [sortOption]);

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    return <DataError />;
  }

  if (totalCount === 0 && !searchQuery) {
    return <EmptyReports isSearching={false} />;
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__controls`}>
        <div className={`${baseClass}__controls-left`}>
          <span className={`${baseClass}__count`}>
            {totalCount} report{totalCount !== 1 ? "s" : ""}
          </span>
          {!saveReportsDisabledInConfig && (
            <Slider
              value={showDontStoreResults}
              onChange={onToggleDontStoreResults}
              activeText="Show reports that don't store results"
              inactiveText="Show reports that don't store results"
              className={`${baseClass}__toggle`}
            />
          )}
        </div>
        <div className={`${baseClass}__controls-right`}>
          <ActionsDropdown
            options={sortDropdownOptions}
            placeholder={`Sort: ${
              SORT_OPTIONS.find((o) => o.value === sortOption)?.label ?? ""
            }`}
            onChange={onSortChange}
            className={`${baseClass}__sort-dropdown`}
            variant="button"
            menuAlign="right"
          />
          <SearchField
            placeholder="Search by name"
            defaultValue={searchQuery}
            onChange={onSearchChange}
          />
        </div>
      </div>

      {reports.length === 0 && searchQuery ? (
        <EmptyReports isSearching />
      ) : (
        <>
          <div className={`${baseClass}__reports-list`}>
            {reports.map((report) => (
              <HostReportCard
                key={report.query_id}
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
            hidePagination={totalCount <= PAGE_SIZE}
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
