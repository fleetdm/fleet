import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { IOperatingSystemVersion } from "interfaces/operating_system";
import {
  getOSVersions,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import LastUpdatedText from "components/LastUpdatedText";
import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import OSVersionTable from "../OSVersionTable";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";
import OSVersionsEmptyState from "../OSVersionsEmptyState";

/** This overrides the `platform` attribute on IOperatingSystemVersion so that only our filtered platforms (currently
 * "darwin" and "windows") values are included */
export type IFilteredOperatingSystemVersion = Omit<
  IOperatingSystemVersion,
  "platform"
> & {
  platform: OSUpdatesSupportedPlatform;
};

const baseClass = "os-updates-current-version-section";

interface ICurrentVersionSectionProps {
  router: InjectedRouter;
  currentTeamId: number;
  queryParams: ReturnType<typeof parseOSUpdatesCurrentVersionsQueryParams>;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 8;

export const parseOSUpdatesCurrentVersionsQueryParams = (queryParams: {
  page?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
}) => {
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;

  return {
    page,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
  };
};

const CurrentVersionSection = ({
  router,
  currentTeamId,
  queryParams,
}: ICurrentVersionSectionProps) => {
  const { data, isError, isLoading: isLoadingOsVersions } = useQuery<
    IOSVersionsResponse,
    AxiosError
  >(
    ["os_versions", currentTeamId, queryParams],
    () => getOSVersions({ teamId: currentTeamId, ...queryParams }),
    {
      retry: false,
      refetchOnWindowFocus: false,
    }
  );

  const generateSubTitleText = () => {
    return (
      <LastUpdatedText
        lastUpdatedAt={data?.counts_updated_at}
        whatToRetrieve="operating systems"
      />
    );
  };

  const renderTable = () => {
    if (isLoadingOsVersions) {
      return <Spinner />;
    }

    if (isError) {
      return (
        <DataError
          className={`${baseClass}__error`}
          description="Refresh the page to try again."
          excludeIssueLink
        />
      );
    }

    if (!data) {
      return null;
    }

    if (!data.os_versions) {
      return <OSVersionsEmptyState />;
    }

    // We only want to show windows mac, ios, ipados versions atm.
    const filteredOSVersionData = data.os_versions.filter((osVersion) => {
      return (
        osVersion.platform === "windows" ||
        osVersion.platform === "darwin" ||
        osVersion.platform === "ios" ||
        osVersion.platform === "ipados"
      );
    }) as IFilteredOperatingSystemVersion[];

    return (
      <OSVersionTable
        router={router}
        osVersionData={filteredOSVersionData}
        currentTeamId={currentTeamId}
        isLoading={isLoadingOsVersions}
        queryParams={queryParams}
        hasNextPage={data.meta.has_next_results}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Current versions"
        subTitle={isLoadingOsVersions ? null : generateSubTitleText()}
        wrapperCustomClass={`${baseClass}__header`}
      />
      {renderTable()}
    </div>
  );
};

export default CurrentVersionSection;
