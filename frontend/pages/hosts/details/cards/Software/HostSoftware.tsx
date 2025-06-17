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
import { HostPlatform, isAndroid, isIPadOrIPhone } from "interfaces/platform";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";
import { convertParamsToSnakeCase } from "utilities/url";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Card from "components/Card/Card";
import CardHeader from "components/CardHeader";
import DataError from "components/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner";
import SoftwareFiltersModal from "pages/SoftwarePage/components/modals/SoftwareFiltersModal";

import {
  buildSoftwareVulnFiltersQueryParams,
  getSoftwareVulnFiltersFromQueryParams,
  ISoftwareVulnFiltersParams,
} from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";
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
  platform: HostPlatform;
  softwareUpdatedAt?: string;
  router: InjectedRouter;
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  pathname: string;
  hostTeamId: number;
  onShowSoftwareDetails: (software: IHostSoftware) => void;
  isSoftwareEnabled?: boolean;
  isMyDevicePage?: boolean;
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
  exploit?: string;
  min_cvss_score?: string;
  max_cvss_score?: string;
  category_id?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const softwareVulnFilters = getSoftwareVulnFiltersFromQueryParams(
    queryParams
  );
  const categoryId = queryParams?.category_id
    ? parseInt(queryParams.category_id, 10)
    : undefined;

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    vulnerable: softwareVulnFilters.vulnerable,
    min_cvss_score: softwareVulnFilters.minCvssScore,
    max_cvss_score: softwareVulnFilters.maxCvssScore,
    exploit: softwareVulnFilters.exploit,
    available_for_install: false, // always false for host software
    category_id: categoryId,
  };
};

const HostSoftware = ({
  id,
  platform,
  softwareUpdatedAt,
  router,
  queryParams,
  pathname,
  hostTeamId = 0,
  onShowSoftwareDetails,
  isSoftwareEnabled = false,
  isMyDevicePage = false,
}: IHostSoftwareProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isPremiumTier,
  } = useContext(AppContext);

  const isUnsupported =
    isAndroid(platform) || (isIPadOrIPhone(platform) && queryParams.vulnerable); // no Android software and no vulnerable software for iOS

  // disables install/uninstall actions after click
  const [softwareIdActionPending, setSoftwareIdActionPending] = useState<
    number | null
  >(null);
  const [showSoftwareFiltersModal, setShowSoftwareFiltersModal] = useState(
    false
  );

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
      enabled: isSoftwareEnabled && !isMyDevicePage && !isUnsupported,
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

  const toggleSoftwareFiltersModal = useCallback(() => {
    setShowSoftwareFiltersModal(!showSoftwareFiltersModal);
  }, [setShowSoftwareFiltersModal, showSoftwareFiltersModal]);

  /**  Compares vuln filters to current vuln query params */
  const determineVulnFilterChange = useCallback(
    (vulnFilters: ISoftwareVulnFiltersParams) => {
      const changedEntry = Object.entries(vulnFilters).find(([key, val]) => {
        switch (key) {
          case "vulnerable":
          case "exploit": {
            // Normalize values: undefined → false, then compare
            const current = queryParams[key] ?? false;
            const incoming = val ?? false;
            return incoming !== current;
          }
          case "minCvssScore":
            return val !== queryParams.min_cvss_score;
          case "maxCvssScore":
            return val !== queryParams.max_cvss_score;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [queryParams]
  );

  const onApplyVulnFilters = (vulnFilters: ISoftwareVulnFiltersParams) => {
    const newQueryParams = {
      query: queryParams.query,
      orderDirection: queryParams.order_direction,
      orderKey: queryParams.order_key,
      perPage: queryParams.per_page,
      page: 0, // resets page index
      ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
    };

    // We want to determine which query param has changed in order to
    // reset the page index to 0 if any other param has changed.
    const changedParam = determineVulnFilterChange(vulnFilters);

    // Update the route only if a change is detected
    if (changedParam) {
      router.replace(
        getNextLocationPath({
          pathPrefix: location.pathname,
          routeTemplate: "",
          queryParams: convertParamsToSnakeCase(newQueryParams),
        })
      );
    }

    toggleSoftwareFiltersModal();
  };

  const tableConfig = useMemo(() => {
    return isMyDevicePage
      ? generateDeviceSoftwareTableConfig()
      : generateHostSoftwareTableConfig({
          router,
          teamId: hostTeamId,
          onClickMoreDetails: onShowSoftwareDetails,
        });
  }, [isMyDevicePage, router, hostTeamId, onShowSoftwareDetails]);

  const isLoading = isMyDevicePage
    ? deviceSoftwareLoading
    : hostSoftwareLoading;

  const isError = isMyDevicePage ? deviceSoftwareError : hostSoftwareError;

  const data = isMyDevicePage ? deviceSoftwareRes : hostSoftwareRes;

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
        {isError &&
          (isMyDevicePage ? (
            <DeviceUserError />
          ) : (
            <DataError verticalPaddingSize="pad-xxxlarge" />
          ))}
        {!isError && (
          <HostSoftwareTable
            isLoading={
              isMyDevicePage ? deviceSoftwareFetching : hostSoftwareFetching
            }
            data={data}
            platform={platform}
            router={router}
            tableConfig={tableConfig}
            sortHeader={queryParams.order_key}
            sortDirection={queryParams.order_direction}
            searchQuery={queryParams.query}
            page={queryParams.page}
            pagePath={pathname}
            vulnFilters={getSoftwareVulnFiltersFromQueryParams(queryParams)}
            onAddFiltersClick={toggleSoftwareFiltersModal}
            pathPrefix={pathname}
            // for my device software details modal toggling
            isMyDevicePage={isMyDevicePage}
            onShowSoftwareDetails={onShowSoftwareDetails}
          />
        )}
        {showSoftwareFiltersModal && (
          <SoftwareFiltersModal
            onExit={toggleSoftwareFiltersModal}
            onSubmit={onApplyVulnFilters}
            vulnFilters={getSoftwareVulnFiltersFromQueryParams(queryParams)}
            isPremiumTier={isPremiumTier || false}
          />
        )}
      </>
    );
  };

  if (isMyDevicePage) {
    return (
      <Card
        className={baseClass}
        borderRadiusSize="xxlarge"
        paddingSize="xlarge"
        includeShadow
      >
        <CardHeader
          header="Software"
          subheader="Software installed on your device."
        />
        {renderHostSoftware()}
      </Card>
    );
  }

  return (
    <div className={baseClass}>
      <CardHeader subheader="Software installed on the host." />
      {renderHostSoftware()}
    </div>
  );
};

export default React.memo(HostSoftware);
