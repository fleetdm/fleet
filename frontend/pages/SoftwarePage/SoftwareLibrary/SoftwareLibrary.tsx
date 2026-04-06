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
} from "services/entities/software";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";

import SoftwareLibraryTable from "./SoftwareLibraryTable";
import {
  ISoftwareDropdownFilterVal,
  buildSoftwareFilterQueryParams,
} from "./SoftwareLibraryTable/helpers";

const baseClass = "software-library";

const DATA_STALE_TIME = 30000;
const QUERY_OPTIONS = {
  keepPreviousData: true,
  staleTime: DATA_STALE_TIME,
};

interface ISoftwareLibraryProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  softwareFilter: ISoftwareDropdownFilterVal;
  currentPage: number;
  teamId?: number;
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
  currentPage,
  teamId,
  addedSoftwareToken,
  onAddFiltersClick,
}: ISoftwareLibraryProps) => {
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

  if (isTitlesLoading) {
    return <Spinner />;
  }

  if (isTitlesError) {
    return <TableDataError verticalPaddingSize="pad-xxxlarge" />;
  }

  return (
    <div className={baseClass}>
      <SoftwareLibraryTable
        router={router}
        data={titlesData}
        isSoftwareEnabled={isSoftwareEnabled}
        query={query}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        softwareFilter={softwareFilter}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={isTitlesFetching}
      />
    </div>
  );
};

export default SoftwareTitles;
