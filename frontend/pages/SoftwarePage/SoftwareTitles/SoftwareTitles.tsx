/**
 software/titles Software tab
 software/versions Software tab (version toggle on)
 */
import React from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { omit } from "lodash";

import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareTitlesQueryKey,
  ISoftwareTitlesResponse,
  ISoftwareVersionsQueryKey,
  ISoftwareVersionsResponse,
} from "services/entities/software";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";

import SoftwareTable from "./SoftwareTable";
import {
  ISoftwareDropdownFilterVal,
  ISoftwareVulnFilters,
  buildSoftwareFilterQueryParams,
} from "./SoftwareTable/helpers";

const baseClass = "software-titles";

const DATA_STALE_TIME = 30000;
const QUERY_OPTIONS = {
  keepPreviousData: true,
  staleTime: DATA_STALE_TIME,
};

interface ISoftwareTitlesProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  softwareFilter: ISoftwareDropdownFilterVal;
  vulnFilters: ISoftwareVulnFilters;
  currentPage: number;
  teamId?: number;
  resetPageIndex: boolean;
  addedSoftwareToken: string | null;
  onAddFiltersClick: () => void;
}

const SoftwareTitles = ({
  router,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  softwareFilter,
  vulnFilters,
  currentPage,
  teamId,
  resetPageIndex,
  addedSoftwareToken,
  onAddFiltersClick,
}: ISoftwareTitlesProps) => {
  const showVersions = location.pathname === PATHS.SOFTWARE_VERSIONS;

  // for Titles view, request to get software data
  const {
    data: titlesData,
    isFetching: isTitlesFetching,
    isLoading: isTitlesLoading,
    isError: isTitlesError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: currentPage,
        perPage,
        query,
        orderDirection,
        orderKey,
        teamId,
        addedSoftwareToken,
        ...vulnFilters,
        ...buildSoftwareFilterQueryParams(softwareFilter),
      },
    ],
    ({ queryKey: [queryKey] }) =>
      softwareAPI.getSoftwareTitles(omit(queryKey, "scope")),
    {
      ...QUERY_OPTIONS,
      enabled: location.pathname === PATHS.SOFTWARE_TITLES,
    }
  );

  // For Versions view, request software versions data. If empty, request titles available for
  // install to determine empty state copy

  const {
    data: versionsData,
    isFetching: isVersionsFetching,
    isLoading: isVersionsLoading,
    isError: isVersionsError,
  } = useQuery<
    ISoftwareVersionsResponse,
    Error,
    ISoftwareVersionsResponse,
    [ISoftwareVersionsQueryKey]
  >(
    [
      {
        scope: "software-versions",
        page: currentPage,
        perPage,
        query,
        orderDirection,
        orderKey,
        teamId,
        addedSoftwareToken,
        ...vulnFilters,
        ...(showVersions ? { without_vulnerability_details: true } : {}),
      },
    ],
    ({ queryKey: [queryKey] }) =>
      softwareAPI.getSoftwareVersions(omit(queryKey, "scope")),
    {
      ...QUERY_OPTIONS,
      enabled: location.pathname === PATHS.SOFTWARE_VERSIONS,
    }
  );

  const {
    data: titlesAvailableForInstallResponse,
    isFetching: isTitlesAFIFetching,
    isLoading: isTitlesAFILoading,
    isError: isTitlesAFIError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: 0,
        perPage,
        query: "",
        orderDirection,
        orderKey,
        teamId,
        availableForInstall: true,
        ...vulnFilters,
      },
    ],
    ({ queryKey: [queryKey] }) =>
      softwareAPI.getSoftwareTitles(omit(queryKey, "scope")),
    {
      ...QUERY_OPTIONS,
      enabled:
        location.pathname === PATHS.SOFTWARE_VERSIONS &&
        versionsData &&
        versionsData.count === 0,
    }
  );

  if (isTitlesLoading || isVersionsLoading || isTitlesAFILoading) {
    return <Spinner />;
  }

  if (isTitlesError || isVersionsError || isTitlesAFIError) {
    return <TableDataError className={`${baseClass}__table-error`} />;
  }

  return (
    <div className={baseClass}>
      <SoftwareTable
        router={router}
        data={showVersions ? versionsData : titlesData}
        showVersions={showVersions}
        installableSoftwareExists={!!titlesAvailableForInstallResponse?.count}
        isSoftwareEnabled={isSoftwareEnabled}
        query={query}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        softwareFilter={softwareFilter}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={
          isTitlesFetching || isVersionsFetching || isTitlesAFIFetching
        }
        resetPageIndex={resetPageIndex}
        onAddFiltersClick={onAddFiltersClick}
        vulnFilters={vulnFilters}
      />
    </div>
  );
};

export default SoftwareTitles;
