import React, { useCallback, useContext, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import hostAPI, {
  IGetHostSoftwareResponse,
  IHostSoftwareQueryKey,
} from "services/entities/hosts";
import deviceAPI, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";
import { IHostSoftware, ISoftware } from "interfaces/software";
import { HostPlatform, isIPadOrIPhone } from "interfaces/platform";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Card from "components/Card/Card";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import { generateSoftwareTableHeaders as generateHostSoftwareTableConfig } from "./HostSoftwareTableConfig";
import { generateSoftwareTableHeaders as generateDeviceSoftwareTableConfig } from "./DeviceSoftwareTableConfig";
import HostSoftwareTable from "./HostSoftwareTable";
import { getInstallErrorMessage, getUninstallErrorMessage } from "./helpers";

const baseClass = "software-card";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface IHostSoftwareProps {
  /** This is the host id or the device token */
  id: number | string;
  platform?: HostPlatform;
  softwareUpdatedAt?: string;
  hostCanWriteSoftware: boolean;
  router: InjectedRouter;
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  pathname: string;
  hostTeamId: number;
  onShowSoftwareDetails?: (software: IHostSoftware) => void;
  isSoftwareEnabled?: boolean;
  hostScriptsEnabled?: boolean;
  isMyDevicePage?: boolean;
  hostMDMEnrolled?: boolean;
}

const DEFAULT_SEARCH_QUERY = "";
const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 20;

export const parseHostSoftwareQueryParams = (queryParams: {
  page?: string;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
  vulnerable?: string;
  available_for_install?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const vulnerable = queryParams.vulnerable === "true";
  const availableForInstall = queryParams.available_for_install === "true";

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    vulnerable,
    available_for_install: availableForInstall,
  };
};

const HostSoftware = ({
  id,
  platform,
  softwareUpdatedAt,
  hostCanWriteSoftware,
  hostScriptsEnabled,
  router,
  queryParams,
  pathname,
  hostTeamId = 0,
  onShowSoftwareDetails,
  isSoftwareEnabled = false,
  isMyDevicePage = false,
  hostMDMEnrolled,
}: IHostSoftwareProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const vulnFilterAndNotSupported =
    isIPadOrIPhone(platform ?? "") && queryParams.vulnerable;
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  // disables install/uninstall actions after click
  const [softwareIdActionPending, setSoftwareIdActionPending] = useState<
    number | null
  >(null);

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
    IHostSoftwareQueryKey[]
  >(
    [
      {
        scope: "host_software",
        id: id as number,
        softwareUpdatedAt,
        ...queryParams,
      },
    ],
    ({ queryKey }) => {
      return hostAPI.getHostSoftware(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled:
        isSoftwareEnabled && !isMyDevicePage && !vulnFilterAndNotSupported,
      keepPreviousData: true,
      staleTime: 7000,
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
    IDeviceSoftwareQueryKey[]
  >(
    [
      {
        scope: "device_software",
        id: id as string,
        softwareUpdatedAt,
        ...queryParams,
      },
    ],
    ({ queryKey }) => deviceAPI.getDeviceSoftware(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled && isMyDevicePage, // if disabled, we'll always show a generic "No software detected" message. No DUP for iPad/iPhone
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const refetchSoftware = useMemo(
    () => (isMyDevicePage ? refetchDeviceSoftware : refetchHostSoftware),
    [isMyDevicePage, refetchDeviceSoftware, refetchHostSoftware]
  );

  const userHasSWWritePermission = Boolean(
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer
  );

  const installHostSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setSoftwareIdActionPending(softwareId);
      try {
        await hostAPI.installHostSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          "Software is installing or will install when the host comes online."
        );
      } catch (e) {
        renderFlash("error", getInstallErrorMessage(e));
      }
      setSoftwareIdActionPending(null);
      refetchSoftware();
    },
    [id, renderFlash, refetchSoftware]
  );

  const uninstallHostSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setSoftwareIdActionPending(softwareId);
      try {
        await hostAPI.uninstallHostSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          <>
            Software is uninstalling or will uninstall when the host comes
            online. To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getUninstallErrorMessage(e));
      }
      setSoftwareIdActionPending(null);
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
        case "uninstall":
          uninstallHostSoftwarePackage(software.id);
          break;
        case "showDetails":
          onShowSoftwareDetails?.(software);
          break;
        default:
          break;
      }
    },
    [
      installHostSoftwarePackage,
      onShowSoftwareDetails,
      uninstallHostSoftwarePackage,
    ]
  );

  const tableConfig = useMemo(() => {
    return isMyDevicePage
      ? generateDeviceSoftwareTableConfig()
      : generateHostSoftwareTableConfig({
          userHasSWWritePermission,
          hostScriptsEnabled,
          hostCanWriteSoftware,
          hostMDMEnrolled,
          softwareIdActionPending,
          router,
          teamId: hostTeamId,
          onSelectAction,
        });
  }, [
    isMyDevicePage,
    router,
    softwareIdActionPending,
    userHasSWWritePermission,
    hostScriptsEnabled,
    onSelectAction,
    hostTeamId,
    hostCanWriteSoftware,
    hostMDMEnrolled,
  ]);

  const isLoading = isMyDevicePage
    ? deviceSoftwareLoading
    : hostSoftwareLoading;

  const isError = isMyDevicePage ? deviceSoftwareError : hostSoftwareError;

  const data = isMyDevicePage ? deviceSoftwareRes : hostSoftwareRes;

  const getHostSoftwareFilterFromQueryParams = () => {
    const { vulnerable, available_for_install } = queryParams;
    if (available_for_install) {
      return "installableSoftware";
    }
    if (vulnerable) {
      return "vulnerableSoftware";
    }
    return "allSoftware";
  };

  const renderHostSoftware = () => {
    if (isLoading) {
      return <Spinner />;
    }
    // will never be the case - to handle `platform` typing discrepancy with DeviceUserPage
    if (!platform) {
      return null;
    }
    return (
      <>
        {isError && <DataError />}
        {!isError && (
          <HostSoftwareTable
            isLoading={
              isMyDevicePage ? deviceSoftwareFetching : hostSoftwareFetching
            }
            // this could be cleaner, however, we are going to revert this commit anyway once vulns are
            // supported for iPad/iPhone, by the end of next sprint
            data={
              vulnFilterAndNotSupported
                ? ({
                    count: 0,
                    meta: {
                      has_next_results: false,
                      has_previous_results: false,
                    },
                  } as IGetHostSoftwareResponse)
                : data
            } // eshould be mpty for iPad/iPhone since API call is disabled, but to be sure to trigger empty state
            platform={platform}
            router={router}
            tableConfig={tableConfig}
            sortHeader={queryParams.order_key}
            sortDirection={queryParams.order_direction}
            searchQuery={queryParams.query}
            page={queryParams.page}
            pagePath={pathname}
            hostSoftwareFilter={getHostSoftwareFilterFromQueryParams()}
            pathPrefix={pathname}
          />
        )}
      </>
    );
  };

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xxlarge"
      includeShadow
      className={baseClass}
    >
      <p className="card__header">Software</p>
      {renderHostSoftware()}
    </Card>
  );
};

export default React.memo(HostSoftware);
