/**
 software/titles Software tab
 software/versions Software tab (version toggle on)
 */
import React from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareApiParams,
  ISoftwareTitlesResponse,
  ISoftwareVersionsResponse,
} from "services/entities/software";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";

import SoftwareTable from "./SoftwareTable";
import { ISoftwareDropdownFilterVal } from "./SoftwareTable/helpers";

const baseClass = "software-titles";

const DATA_STALE_TIME = 30000;
const QUERY_OPTIONS = {
  keepPreviousData: true,
  staleTime: DATA_STALE_TIME,
};

interface ISoftwareTitlesQueryKey extends ISoftwareApiParams {
  scope: "software-titles";
}

interface ISoftwareVersionsQueryKey extends ISoftwareApiParams {
  scope: "software-versions";
}

interface ISoftwareTitlesProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  softwareFilter: ISoftwareDropdownFilterVal;
  currentPage: number;
  teamId?: number;
  resetPageIndex: boolean;
}

const SoftwareTitles = ({
  router,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  softwareFilter,
  currentPage,
  teamId,
  resetPageIndex,
}: ISoftwareTitlesProps) => {
  const showVersions = location.pathname === PATHS.SOFTWARE_VERSIONS;

  const generateSoftwareTitlesQueryKey = (): ISoftwareTitlesQueryKey => {
    const queryKey: ISoftwareTitlesQueryKey = {
      scope: "software-titles",
      page: currentPage,
      perPage,
      query,
      orderDirection,
      orderKey,
      teamId,
    };
    if (softwareFilter === "installableSoftware") {
      queryKey.availableForInstall = true;
    } else {
      queryKey.vulnerable = softwareFilter === "vulnerableSoftware";
    }

    return queryKey;
  };

  // request to get software data
  const {
    data: titlesData,
    isFetching: isTitlesFetching,
    isLoading: isTitlesLoading,
    isError: isTitlesError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    ISoftwareTitlesQueryKey[]
  >(
    [generateSoftwareTitlesQueryKey()],
    ({ queryKey }) => softwareAPI.getSoftwareTitles(queryKey[0]),
    {
      ...QUERY_OPTIONS,
      enabled: location.pathname === PATHS.SOFTWARE_TITLES,
    }
  );

  // request to get software versions data
  const {
    data: versionsData,
    isFetching: isVersionsFetching,
    isLoading: isVersionsLoading,
    isError: isVersionsError,
  } = useQuery<
    ISoftwareVersionsResponse,
    Error,
    ISoftwareVersionsResponse,
    ISoftwareVersionsQueryKey[]
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
        vulnerable: softwareFilter === "vulnerableSoftware",
      },
    ],
    ({ queryKey }) => softwareAPI.getSoftwareVersions(queryKey[0]),
    {
      ...QUERY_OPTIONS,
      enabled: location.pathname === PATHS.SOFTWARE_VERSIONS,
    }
  );

  if (isTitlesLoading || isVersionsLoading) {
    return <Spinner />;
  }

  if (isTitlesError || isVersionsError) {
    return <TableDataError className={`${baseClass}__table-error`} />;
  }

  return (
    <div className={baseClass}>
      <SoftwareTable
        router={router}
        data={showVersions ? versionsData : titlesData}
        showVersions={showVersions}
        isSoftwareEnabled={isSoftwareEnabled}
        query={query}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        softwareFilter={softwareFilter}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={isTitlesFetching || isVersionsFetching}
        resetPageIndex={resetPageIndex}
      />
    </div>
  );
};

export default SoftwareTitles;
