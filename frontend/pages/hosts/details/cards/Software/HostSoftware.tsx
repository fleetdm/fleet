import { AxiosError } from "axios";
import { isEqual, omit } from "lodash";
import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import CardHeader from "components/CardHeader";
import DataError from "components/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import {
  HostPlatform,
  isAndroid,
  isIPadOrIPhone,
  isMacOS,
} from "interfaces/platform";
import {
  getSoftwareTypesForPlatform,
  IHostSoftware,
  ISoftware,
  MACOS_APP_SOFTWARE_TYPE,
  softwareTypesToApiParams,
} from "interfaces/software";
import SoftwareFiltersModal from "pages/SoftwarePage/components/modals/SoftwareFiltersModal";
import {
  buildSoftwareFiltersQueryParams,
  getSoftwareFiltersFromQueryParams,
  ISoftwareFilters,
} from "pages/SoftwarePage/SoftwareInventory/SoftwareInventoryTable/helpers";
import deviceAPI, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";
import hostAPI, {
  IGetHostSoftwareResponse,
  IHostSoftwareQueryKey,
} from "services/entities/hosts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { generateSoftwareTableHeaders as generateDeviceSoftwareTableConfig } from "./DeviceSoftwareTableConfig";
import { getHostSoftwareLocationPath, getSoftwareSubheader } from "./helpers";
import HostSoftwareTable from "./HostSoftwareTable";
import { generateSoftwareTableHeaders as generateHostSoftwareTableConfig } from "./HostSoftwareTableConfig";

const baseClass = "host-software-section";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface HostSoftwareQueryParams
  extends ReturnType<typeof parseHostSoftwareQueryParams> {
  include_available_for_install?: boolean;
}

interface IHostSoftwareProps {
  /** This is the host id or the device token */
  id: number | string;
  platform: HostPlatform;
  softwareUpdatedAt?: string;
  router: InjectedRouter;
  queryParams: HostSoftwareQueryParams;
  pathname: string;
  hostTeamId: number;
  onShowInventoryVersions: (software: IHostSoftware) => void;
  isSoftwareEnabled?: boolean;
  isMyDevicePage?: boolean;
  /**
   * Premium status for the My device page. The device page is token-authenticated
   * and has no app session, so `isPremiumTier` is not available from the app
   * context there and must be passed in explicitly from the device's license info.
   * Ignored on the host details page, which reads premium status from the app context.
   */
  isPremiumTier?: boolean;
  /** Used to show custom Software card header */
  hostMdmEnrollmentStatus?: MdmEnrollmentStatus | null;
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
  self_service?: string;
  category_id?: string;
  fleet_id?: string;
  macos_applications?: string;
  types?: string;
  ai_tool?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const softwareFilters = getSoftwareFiltersFromQueryParams(queryParams);
  const categoryId = queryParams?.category_id
    ? parseInt(queryParams.category_id, 10)
    : undefined;
  const selfService = queryParams?.self_service === "true";
  const teamId = queryParams?.fleet_id
    ? parseInt(queryParams.fleet_id, 10)
    : undefined;
  // Tri-state: true/false when explicitly set in the URL, otherwise undefined so
  // the platform-specific default can be applied where the platform is known.
  let macosApplications: boolean | undefined;
  if (queryParams?.macos_applications === "true") {
    macosApplications = true;
  } else if (queryParams?.macos_applications === "false") {
    macosApplications = false;
  }

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    vulnerable: softwareFilters.vulnerable,
    min_cvss_score: softwareFilters.minCvssScore,
    max_cvss_score: softwareFilters.maxCvssScore,
    self_service: selfService,
    exploit: softwareFilters.exploit,
    available_for_install: false, // always false for host software
    category_id: categoryId,
    fleet_id: teamId,
    macos_applications: macosApplications,
    types: softwareFilters.types,
    ai_tool: softwareFilters.aiTool,
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
  onShowInventoryVersions,
  isSoftwareEnabled = false,
  isMyDevicePage = false,
  isPremiumTier: isPremiumTierProp,
  hostMdmEnrollmentStatus = null,
}: IHostSoftwareProps) => {
  const { isPremiumTier: isPremiumTierFromContext } = useContext(AppContext);
  // The My device page is token-authenticated and has no app session/context, so
  // its premium status is provided explicitly by the caller. Everywhere else we
  // read it from the app context.
  const isPremiumTier = isMyDevicePage
    ? isPremiumTierProp
    : isPremiumTierFromContext;

  const availableTypes = useMemo(
    () =>
      getSoftwareTypesForPlatform(platform, {
        hostPage: true,
        premium: isPremiumTier,
      }),
    [platform, isPremiumTier]
  );

  // Keys for another platform (a URL copied between hosts) or, on Free, for a
  // Premium-only type are dropped. They would otherwise filter the list with no
  // chip in the picker to clear them.
  const selectedTypes = (queryParams.types ?? []).filter((key) =>
    availableTypes.some((t) => t.key === key)
  );

  // The URL is the source of truth for the selection: foreign keys are
  // removed, and a macOS host whose URL carries no applicable type gets
  // "macOS app" unless it says `types=none` (the user cleared the selection).
  // Tab navigation strips the query string, so this runs on every URL change.
  const isCleared = queryParams.types?.length === 0;
  const normalizedTypes =
    isMacOS(platform) && selectedTypes.length === 0 && !isCleared
      ? [MACOS_APP_SOFTWARE_TYPE]
      : selectedTypes;
  const isNormalizingUrl = !isEqual(queryParams.types ?? [], normalizedTypes);

  // "Show helpers" is the inverse of the /Applications filter and only applies
  // while "macOS app" is selected.
  const macosApplicationsFilter =
    isMacOS(platform) && selectedTypes.includes(MACOS_APP_SOFTWARE_TYPE)
      ? queryParams.macos_applications ?? true
      : undefined;

  const filters: ISoftwareFilters = {
    vulnerable: queryParams.vulnerable,
    exploit: queryParams.exploit,
    minCvssScore: queryParams.min_cvss_score,
    maxCvssScore: queryParams.max_cvss_score,
    types: selectedTypes,
    // Free rejects ai_tool, which a URL copied from a Premium server can carry.
    aiTool: (isPremiumTier && queryParams.ai_tool) || undefined,
  };

  const apiQueryParams = {
    ...omit(queryParams, "types"),
    ...softwareTypesToApiParams(selectedTypes),
    // Show helpers on means no pruning, so the param is omitted.
    macos_applications: macosApplicationsFilter || undefined,
    ai_tool: filters.aiTool,
  };

  // no Android software and no vulnerable software for iOS
  const isUnsupported = isIPadOrIPhone(platform) && queryParams.vulnerable;

  const [showSoftwareFiltersModal, setShowSoftwareFiltersModal] = useState(
    false
  );

  const {
    data: hostSoftwareRes,
    isLoading: hostSoftwareLoading,
    isError: hostSoftwareError,
    isFetching: hostSoftwareFetching,
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
        ...apiQueryParams,
      },
    ],
    ({ queryKey }) => {
      return hostAPI.getHostSoftware(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled:
        isSoftwareEnabled &&
        !isMyDevicePage &&
        !isUnsupported &&
        !isNormalizingUrl,
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const {
    data: deviceSoftwareRes,
    isLoading: deviceSoftwareLoading,
    isError: deviceSoftwareError,
    isFetching: deviceSoftwareFetching,
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
        ...apiQueryParams,
      },
    ],
    ({ queryKey }) => deviceAPI.getDeviceSoftware(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled && isMyDevicePage && !isNormalizingUrl, // if disabled, we'll always show a generic "No software detected" message. No My Device Page for iPad/iPhone
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const toggleSoftwareFiltersModal = useCallback(() => {
    setShowSoftwareFiltersModal(!showSoftwareFiltersModal);
  }, [setShowSoftwareFiltersModal, showSoftwareFiltersModal]);

  const getFilteredLocationPath = (
    newFilters: ISoftwareFilters,
    page: number
  ) =>
    getHostSoftwareLocationPath({
      pathname,
      platform,
      query: queryParams.query,
      orderKey: queryParams.order_key,
      orderDirection: queryParams.order_direction,
      page,
      fleetId: queryParams.fleet_id,
      // Preserve an explicit Show helpers selection only while "macOS app"
      // stays selected, so reselecting it brings the toggle back off.
      macosApplications: newFilters.types?.includes(MACOS_APP_SOFTWARE_TYPE)
        ? queryParams.macos_applications
        : undefined,
      filters: newFilters,
    });

  const normalizedPath = isNormalizingUrl
    ? getFilteredLocationPath(
        { ...filters, types: normalizedTypes },
        queryParams.page
      )
    : undefined;

  useEffect(() => {
    if (normalizedPath) router.replace(normalizedPath);
  }, [normalizedPath, router]);

  const onApplyFilters = (newFilters: ISoftwareFilters) => {
    // Leave the URL (and page index) alone when the filters didn't change.
    if (
      !isEqual(
        buildSoftwareFiltersQueryParams(newFilters),
        buildSoftwareFiltersQueryParams(filters)
      )
    ) {
      router.replace(getFilteredLocationPath(newFilters, 0));
    }

    toggleSoftwareFiltersModal();
  };

  const tableConfig = useMemo(() => {
    return isMyDevicePage
      ? generateDeviceSoftwareTableConfig()
      : generateHostSoftwareTableConfig({
          router,
          teamId: hostTeamId,
          onShowInventoryVersions,
        });
  }, [isMyDevicePage, router, hostTeamId, onShowInventoryVersions]);

  const isLoading =
    isNormalizingUrl ||
    (isMyDevicePage ? deviceSoftwareLoading : hostSoftwareLoading);

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
            filters={filters}
            teamId={queryParams.fleet_id}
            macosApplicationsFilter={macosApplicationsFilter}
            onAddFiltersClick={toggleSoftwareFiltersModal}
            // for my device software details modal toggling
            isMyDevicePage={isMyDevicePage}
            onShowInventoryVersions={onShowInventoryVersions}
          />
        )}
        {showSoftwareFiltersModal && (
          <SoftwareFiltersModal
            onExit={toggleSoftwareFiltersModal}
            onSubmit={onApplyFilters}
            filters={filters}
            isPremiumTier={isPremiumTier || false}
            availableTypes={availableTypes}
            showAiToolFilter
          />
        )}
      </>
    );
  };

  if (isMyDevicePage) {
    return (
      <div className={baseClass}>
        <CardHeader
          header="Software"
          subheader={
            // Fleet Free does not have card subheader
            isPremiumTier
              ? getSoftwareSubheader({
                  platform,
                  isMyDevicePage: true,
                  hostMdmEnrollmentStatus,
                })
              : undefined
          }
        />
        {renderHostSoftware()}
      </div>
    );
  }

  return (
    <div className={baseClass}>
      {/* Fleet Free and Android both do not have card subheader */}
      {!isAndroid(platform) && isPremiumTier && (
        <CardHeader
          subheader={getSoftwareSubheader({
            platform,
            isMyDevicePage: false,
            hostMdmEnrollmentStatus,
          })}
        />
      )}
      {renderHostSoftware()}
    </div>
  );
};

// TODO - name this consistently, it is confusing. This same component is called `SoftwareInventoryCard` one place,
// `SoftwareCard` another, and `HostSoftware` here.
export default React.memo(HostSoftware);
