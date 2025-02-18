/** software/os OS tab */

import React from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import {
  IGetOSVersionsQueryKey,
  IOSVersionsResponse,
  getOSVersions,
} from "services/entities/operating_systems";

import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import { SelectedPlatform } from "interfaces/platform";

import SoftwareOSTable from "./SoftwareOSTable";

const baseClass = "software-os";

interface ISoftwareOSProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  currentPage: number;
  teamId?: number;
  resetPageIndex: boolean;
  platform: SelectedPlatform;
}

const SoftwareOS = ({
  router,
  isSoftwareEnabled,
  perPage,
  orderDirection,
  orderKey,
  currentPage,
  teamId,
  resetPageIndex,
  platform,
}: ISoftwareOSProps) => {
  const queryParams = {
    page: currentPage,
    per_page: perPage,
    order_direction: orderDirection,
    order_key: orderKey,
    platform: platform === "all" ? undefined : platform,
    teamId,
  };

  const { data, isFetching, isLoading, isError } = useQuery<
    IOSVersionsResponse,
    Error,
    IOSVersionsResponse,
    IGetOSVersionsQueryKey[]
  >(
    [
      {
        scope: "software-os",
        ...queryParams,
      },
    ],
    () => getOSVersions(queryParams),
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    return <TableDataError className={`${baseClass}__table-error`} />;
  }

  return (
    <div className={baseClass}>
      <SoftwareOSTable
        router={router}
        data={data}
        isSoftwareEnabled={isSoftwareEnabled}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={isFetching}
        resetPageIndex={resetPageIndex}
        platform={platform}
      />
    </div>
  );
};

export default SoftwareOS;
