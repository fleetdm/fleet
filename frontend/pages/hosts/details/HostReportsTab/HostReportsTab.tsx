import React, { useState, useCallback, useMemo } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import hostReportsAPI, {
  IHostReport,
  IListHostReportsResponse,
} from "services/entities/host_reports";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SearchField from "components/forms/fields/SearchField";
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
  saveReportsDisabledInConfig?: boolean;
}

const HostReportsTab = ({
  hostId,
  hostName,
  router,
  saveReportsDisabledInConfig,
}: IHostReportsTabProps): JSX.Element => {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortOption, setSortOption] = useState<SortOption>("newest_results");
  const [page, setPage] = useState(0);
  const [showDontStoreResults, setShowDontStoreResults] = useState(false);

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
      setSearchQuery(value);
      setPage(0);
    },
    [setSearchQuery, setPage]
  );

  const onSortChange = useCallback(
    (value: string) => {
      setSortOption(value as SortOption);
      setPage(0);
    },
    [setSortOption, setPage]
  );

  const onToggleDontStoreResults = useCallback(() => {
    setShowDontStoreResults((prev) => !prev);
    setPage(0);
  }, []);

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

  // Empty state: no reports at all (no search active)
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
            <label
              className={`${baseClass}__toggle`}
              htmlFor="show-dont-store-results"
            >
              <input
                id="show-dont-store-results"
                type="checkbox"
                checked={showDontStoreResults}
                onChange={onToggleDontStoreResults}
              />
              Show reports that don&apos;t store results
            </label>
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
          <SearchField placeholder="Search by name" onChange={onSearchChange} />
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
                hostId={hostId}
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
