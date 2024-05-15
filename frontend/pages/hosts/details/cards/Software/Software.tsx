import React, { useCallback, useContext, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { trimEnd, upperFirst } from "lodash";

import hostAPI, {
  IGetHostSoftwareResponse,
  IHostSoftwareQueryParams,
} from "services/entities/hosts";
import deviceAPI, {
  IDeviceSoftwareQueryParams,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";
import { getErrorReason } from "interfaces/errors";
import { IHostSoftware, ISoftware } from "interfaces/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Card from "components/Card";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import { generateSoftwareTableHeaders as generateHostSoftwareTableConfig } from "./HostSoftwareTableConfig";
import { generateSoftwareTableHeaders as generateDeviceSoftwareTableConfig } from "./DeviceSoftwareTableConfig";
import HostSoftwareTable from "./HostSoftwareTable";

const baseClass = "software-card";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface ISoftwareCardProps {
  /** This is the host id or the device token */
  id: number | string;
  isFleetdHost: boolean;
  router: InjectedRouter;
  queryParams?: {
    page?: string;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
  };
  pathname: string;
  /** Team id for the host */
  teamId: number;
  onShowSoftwareDetails?: (software: IHostSoftware) => void;
  isSoftwareEnabled?: boolean;
  isMyDevicePage?: boolean;
}

const DEFAULT_SEARCH_QUERY = "";
const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 20;

const SoftwareCard = ({
  id,
  isFleetdHost,
  router,
  queryParams,
  pathname,
  teamId = 0,
  onShowSoftwareDetails,
  isSoftwareEnabled = false,
  isMyDevicePage = false,
}: ISoftwareCardProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  const [installingSoftwareId, setInstallingSoftwareId] = useState<
    number | null
  >(null);

  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;

  const {
    data: hostSoftwareRes,
    isLoading: hostSoftwareLoading,
    isError: hostSoftwareError,
    isFetching: hostSoftwareFetching,
    refetch: refetchHostSoftware,
  } = useQuery<
    IGetHostSoftwareResponse,
    AxiosError,
    IGetHostSoftwareResponse,
    [string, IHostSoftwareQueryParams]
  >(
    [
      "host-software",
      {
        page,
        per_page: pageSize,
        query: searchQuery,
        order_key: sortHeader,
        order_direction: sortDirection,
      },
    ],
    ({ queryKey }) => {
      return hostAPI.getHostSoftware(id as number, queryKey[1]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled && !isMyDevicePage,
    }
  );

  const {
    data: deviceSoftwareRes,
    isLoading: deviceSoftwareLoading,
    isError: deviceSoftwareError,
    isFetching: deviceSoftwareFetching,
    refetch: refetchDeviceSoftware,
  } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError,
    IGetDeviceSoftwareResponse,
    [string, IDeviceSoftwareQueryParams]
  >(
    [
      "device-software",
      {
        page,
        per_page: pageSize,
        query: searchQuery,
        order_key: sortHeader,
        order_direction: sortDirection,
      },
    ],
    ({ queryKey }) => deviceAPI.getDeviceSoftware(id as string, queryKey[1]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled && isMyDevicePage,
    }
  );

  const refetchSoftware = useMemo(
    () => (isMyDevicePage ? refetchDeviceSoftware : refetchHostSoftware),
    [isMyDevicePage, refetchDeviceSoftware, refetchHostSoftware]
  );

  const canInstallSoftware = Boolean(
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer
  );

  const installHostSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setInstallingSoftwareId(softwareId);
      try {
        await hostAPI.installHostSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          "Software is installing or will install when the host comes online."
        );
      } catch (e) {
        const reason = upperFirst(trimEnd(getErrorReason(e), "."));
        if (reason.includes("fleetd installed")) {
          renderFlash("error", `Couldn't install. ${reason}.`);
        } else if (reason.includes("can be installed only on")) {
          renderFlash(
            "error",
            `Couldn't install. ${reason.replace("darwin", "macOS")}.`
          );
        } else {
          renderFlash("error", "Couldn't install. Please try again.");
        }
      }
      setInstallingSoftwareId(null);
      refetchSoftware();
    },
    [id, renderFlash, refetchSoftware]
  );

  const onSelectAction = useCallback(
    (software: IHostSoftware, action: string) => {
      switch (action) {
        case "install":
          installHostSoftwarePackage(software.id);
          break;
        case "showDetails":
          onShowSoftwareDetails?.(software);
          break;
        default:
          break;
      }
    },
    [installHostSoftwarePackage, onShowSoftwareDetails]
  );

  const tableConfig = useMemo(() => {
    return isMyDevicePage
      ? generateDeviceSoftwareTableConfig()
      : generateHostSoftwareTableConfig({
          router,
          installingSoftwareId,
          canInstall: canInstallSoftware,
          onSelectAction,
          teamId,
          isFleetdHost,
        });
  }, [
    isMyDevicePage,
    router,
    installingSoftwareId,
    canInstallSoftware,
    onSelectAction,
    teamId,
    isFleetdHost,
  ]);

  const renderSoftwareTable = () => {
    if (hostSoftwareLoading || deviceSoftwareLoading) {
      return <Spinner />;
    }

    if (hostSoftwareError || deviceSoftwareError) {
      return <DataError />;
    }

    const props = {
      router,
      tableConfig,
      sortHeader,
      sortDirection,
      searchQuery,
      page,
      pagePath: pathname,
    };

    if (!isMyDevicePage) {
      return hostSoftwareRes ? (
        <HostSoftwareTable
          isLoading={hostSoftwareLoading}
          data={hostSoftwareRes}
          {...props}
        />
      ) : null;
    }

    return deviceSoftwareRes ? (
      <HostSoftwareTable
        isLoading={deviceSoftwareLoading}
        data={deviceSoftwareRes}
        {...props}
      />
    ) : null;
  };

  return (
    <Card
      borderRadiusSize="large"
      includeShadow
      largePadding
      className={baseClass}
    >
      <p className="card__header">Software</p>
      {renderSoftwareTable()}
    </Card>
  );
};
export default React.memo(SoftwareCard);
