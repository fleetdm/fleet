// TODO: make 'queryParams', 'router', and 'tableQueryData' dependencies stable (aka, memoized)

import { isEqual } from "lodash";
import React, {
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useQuery, useQueryClient } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import { SingleValue } from "react-select-5";

import AutomationsButton from "components/buttons/AutomationsButton";
import Button from "components/buttons/Button";
import TableDataError from "components/DataError";
import FleetsDropdown from "components/FleetsDropdown";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import LastUpdatedText from "components/LastUpdatedText";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IConfig } from "interfaces/config";
import { isQueryablePlatform } from "interfaces/platform";
import {
  IPolicyStats,
  ILoadAllPoliciesResponse,
  IPoliciesCountResponse,
  OtherAutomationType,
} from "interfaces/policy";
import {
  API_ALL_TEAMS_ID,
  API_NO_TEAM_ID,
  APP_CONTEXT_ALL_TEAMS_ID,
  ITeamConfig,
} from "interfaces/team";
import { getTicketOrWebhookState } from "pages/policies/helpers";
import PATHS from "router/paths";
import configAPI from "services/entities/config";
import policiesAPI, {
  IPoliciesCountQueryKey,
  IPoliciesQueryKey,
} from "services/entities/policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { getNextLocationPath } from "utilities/helpers";

import { DEFAULT_POLICY } from "../constants";

import AutomationsModal from "./components/AutomationsModal";
import DeletePoliciesModal from "./components/DeletePoliciesModal";
import ManageAutomationsModal from "./components/ManageAutomationsModal";
import PoliciesTable from "./components/PoliciesTable";
import { getPoliciesFilterParams, PoliciesAutomationFilter } from "./helpers";

interface IManagePoliciesPageProps {
  router: InjectedRouter;
  location: {
    action: string;
    hash: string;
    key: string;
    pathname: string;
    query: {
      fleet_id?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      page?: string;
      automation_type?: PoliciesAutomationFilter;
      platform?: string;
      manage_automations?: string;
    };
    search: string;
  };
}

export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 20;
export const DEFAULT_SORT_COLUMN = "name";

const AUTOMATION_TYPES: PoliciesAutomationFilter[] = [
  "software",
  "patch",
  "script",
  "configuration_profile",
  "calendar",
  "conditional_access",
  "webhook_or_ticket",
];

const GLOBAL_AUTOMATION_TYPES: PoliciesAutomationFilter[] = [
  "webhook_or_ticket",
];

const getValidAutomationTypesForTeam = (
  teamIdForApi: number | undefined
): PoliciesAutomationFilter[] => {
  if (teamIdForApi === undefined) {
    // All fleets → global policies only support webhook/ticket automations.
    return GLOBAL_AUTOMATION_TYPES;
  }
  if (teamIdForApi === API_NO_TEAM_ID) {
    // Unassigned supports every automation type EXCEPT calendar events,
    // which PolicyAutomationsFields hardcodes as fleet-only (never available
    // for "All fleets" or "Unassigned").
    return AUTOMATION_TYPES.filter((type) => type !== "calendar");
  }
  return AUTOMATION_TYPES;
};

const baseClass = "manage-policies-page";

const ManagePolicyPage = ({
  router,
  location,
}: IManagePoliciesPageProps): JSX.Element => {
  const queryParams = location.query;
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isPremiumTier,
    config: globalConfigFromContext,
    setConfig,
    setFilteredPoliciesPath,
    filteredPoliciesPath,
  } = useContext(AppContext);
  const isPrimoMode =
    globalConfigFromContext?.partnerships?.enable_primo || false;

  const { setResetSelectedRows } = useContext(TableContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryHidden,
    setLastEditedQueryPlatform,
    setLastEditedQueryBody,
    setLastEditedQueryId,
    setPolicyTeamId,
  } = useContext(PolicyContext);

  const {
    currentTeamId,
    currentTeamSummary,
    isAllTeamsSelected,
    isTeamAdmin,
    isTeamMaintainer,
    isRouteOk,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: true,
      observer_plus: true,
      technician: true,
    },
  });

  // loading state used by various policy updates on this page
  const [isUpdatingPolicies, setIsUpdatingPolicies] = useState(false);

  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showDeletePoliciesModal, setShowDeletePoliciesModal] = useState(false);
  const [showAutomationsModal, setShowAutomationsModal] = useState(false);
  const [
    selectedPolicyForAutomations,
    setSelectedPolicyForAutomations,
  ] = useState<IPolicyStats | null>(null);
  // Functions to avoid race conditions
  const initialSearchQuery = (() => queryParams.query ?? "")();
  const initialSortHeader = (() =>
    (queryParams?.order_key as "name" | "failing_host_count") ??
    DEFAULT_SORT_COLUMN)();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ??
    DEFAULT_SORT_DIRECTION)();
  const page =
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0;
  const targetedPlatformParam = isQueryablePlatform(queryParams?.platform)
    ? queryParams?.platform
    : undefined;
  const initialAutomationFilter = (() => {
    const automationQueryParam = queryParams.automation_type;

    if (!automationQueryParam) {
      return null;
    }

    const validValues = getValidAutomationTypesForTeam(teamIdForApi);

    return validValues.includes(automationQueryParam)
      ? automationQueryParam
      : null;
  })();

  const isFirstNavigation = useRef(true);

  // Needs update on location change or table state might not match URL
  const [searchQuery, setSearchQuery] = useState(initialSearchQuery);
  const [
    tableQueryDataForApi,
    setTableQueryDataForApi,
  ] = useState<ITableQueryData>();
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(initialSortDirection);
  const [
    automationFilter,
    setAutomationFilter,
  ] = useState<PoliciesAutomationFilter | null>(initialAutomationFilter);
  const filterParams = getPoliciesFilterParams(automationFilter);

  useEffect(() => {
    setLastEditedQueryPlatform(null);
  }, [setLastEditedQueryPlatform]);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    setSearchQuery(initialSearchQuery);
    setSortHeader(initialSortHeader);
    setSortDirection(initialSortDirection);
    setAutomationFilter(initialAutomationFilter);
  }, [
    location,
    isRouteOk,
    initialSearchQuery,
    initialSortHeader,
    initialSortDirection,
    initialAutomationFilter,
  ]);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    const path = location.pathname + location.search;
    // udpate app context with URL path
    if (location.search && filteredPoliciesPath !== path) {
      setFilteredPoliciesPath(path);
    }
  }, [
    location.pathname,
    location.search,
    filteredPoliciesPath,
    setFilteredPoliciesPath,
    isRouteOk,
  ]);

  const {
    data: globalPolicies,
    error: globalPoliciesError,
    isFetching: isFetchingGlobalPolicies,
    refetch: refetchGlobalPolicies,
  } = useQuery<
    ILoadAllPoliciesResponse,
    Error,
    IPolicyStats[],
    IPoliciesQueryKey[]
  >(
    [
      {
        scope: "policies",
        page,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        ...filterParams,
        platform: targetedPlatformParam,
      },
    ],
    ({ queryKey }) => policiesAPI.loadAll(queryKey[0]),
    {
      enabled: isRouteOk && isAllTeamsSelected,
      select: (data) => data.policies || [],
      staleTime: 5000,
      refetchOnWindowFocus: false,
    }
  );

  const {
    data: globalPoliciesCount,
    isFetching: isFetchingGlobalCount,
    isError: isErrorGlobalPoliciesCount,
    refetch: refetchGlobalPoliciesCount,
  } = useQuery<IPoliciesCountResponse, Error, number, IPoliciesCountQueryKey[]>(
    [
      {
        scope: "policiesCount",
        query: !isAllTeamsSelected ? "" : searchQuery,
        ...filterParams,
        platform: targetedPlatformParam,
      },
    ],
    ({ queryKey }) => policiesAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && isAllTeamsSelected,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  const {
    data: teamPolicies,
    error: teamPoliciesError,
    isFetching: isFetchingTeamPolicies,
    refetch: refetchTeamPolicies,
  } = useQuery<
    ILoadAllPoliciesResponse,
    Error,
    IPolicyStats[],
    IPoliciesQueryKey[]
  >(
    [
      {
        scope: "policies",
        page,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        // teamIdForApi will never actually be undefined here
        fleetId: teamIdForApi || 0,
        // no teams does inherit
        mergeInherited: true,
        ...filterParams,
        platform: targetedPlatformParam,
      },
    ],
    ({ queryKey }) => policiesAPI.loadAll(queryKey[0]),
    {
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
      select: (data: ILoadAllPoliciesResponse) => data.policies || [],
      refetchOnWindowFocus: false,
    }
  );

  const {
    data: teamPoliciesCountMergeInherited,
    isFetching: isFetchingTeamCountMergeInherited,
    refetch: refetchTeamPoliciesCountMergeInherited,
  } = useQuery<IPoliciesCountResponse, Error, number, IPoliciesCountQueryKey[]>(
    [
      {
        scope: "policiesCount",
        query: searchQuery,
        fleetId: teamIdForApi || 0, // TODO: Fix number/undefined type
        mergeInherited: true,
        ...filterParams,
        platform: targetedPlatformParam,
      },
    ],
    ({ queryKey }) => policiesAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  // The fleet's own policies, without the inherited ones, which decide
  // whether there's anything to automate or delete here.
  const {
    data: teamOwnPoliciesCountData,
    isError: isErrorTeamOwnPoliciesCount,
    isPreviousData: isPreviousTeamOwnPoliciesCount,
    refetch: refetchTeamOwnPoliciesCount,
  } = useQuery<IPoliciesCountResponse, Error, number, IPoliciesCountQueryKey[]>(
    [
      {
        scope: "policiesCount",
        query: searchQuery,
        fleetId: teamIdForApi || 0,
        ...filterParams,
        platform: targetedPlatformParam,
      },
    ],
    ({ queryKey }) => policiesAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  // keepPreviousData can still hold another fleet's count; don't use it as
  // this fleet's.
  const teamOwnPoliciesCount = isPreviousTeamOwnPoliciesCount
    ? undefined
    : teamOwnPoliciesCountData;

  const canAddOrDeletePolicies =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;

  const canEditAutomationsSettings = isGlobalAdmin || isTeamAdmin;

  const { data: globalConfig, isFetching: isFetchingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(
    ["config"],
    () => {
      return configAPI.loadAll();
    },
    {
      enabled: isRouteOk,
      onSuccess: (data) => {
        setConfig(data);
      },
      staleTime: 5000,
      refetchOnWindowFocus: false,
    }
  );

  const { data: teamData, isFetching: isFetchingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error
  >(["teams", teamIdForApi], () => teamsAPI.load(teamIdForApi), {
    // Enable for all teams including "No team" (teamIdForApi === 0)
    enabled: isRouteOk && teamIdForApi !== undefined,
    staleTime: 5000,
    refetchOnWindowFocus: false,
  });
  const teamConfig = teamData?.fleet;

  const automationsConfig = isAllTeamsSelected ? globalConfig : teamConfig;

  const refetchPolicies = (teamId?: number) => {
    if (teamId !== undefined) {
      refetchTeamPolicies();
      refetchTeamPoliciesCountMergeInherited();
      refetchTeamOwnPoliciesCount();
    } else {
      refetchGlobalPolicies(); // Only call on global policies as this is expensive
      refetchGlobalPoliciesCount();
    }
  };

  const queryClient = useQueryClient();

  // Fleet and Unassigned lists include inherited policies, so any scope's
  // cached list or count can hold a deleted policy. Cached scopes not on
  // screen are dropped, since a stale cache would still show the row while it
  // refetches.
  const invalidateAllPolicies = useCallback(() => {
    ["policies", "policiesCount"].forEach((scope) => {
      queryClient.removeQueries([{ scope }], { inactive: true });
      queryClient.invalidateQueries([{ scope }]);
    });
  }, [queryClient]);

  // Browser Back changes the fleet without onTeamChange; a delete
  // confirmation must not carry another fleet's selection.
  useEffect(() => {
    setShowDeletePoliciesModal(false);
    setSelectedPolicyIds([]);
  }, [teamIdForApi]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      setSelectedPolicyIds([]);
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      if (!isRouteOk || isEqual(newTableQuery, tableQueryDataForApi)) {
        return;
      }

      setTableQueryDataForApi({ ...newTableQuery });

      const {
        pageIndex: newPageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
        sortHeader: newSortHeader,
      } = newTableQuery;
      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};

      newQueryParams.query = newSearchQuery;

      newQueryParams.order_key = newSortHeader;
      newQueryParams.order_direction = newSortDirection;
      newQueryParams.page = newPageIndex.toString();

      // Reset page number to 0 for new filters
      if (
        newSortDirection !== sortDirection ||
        newSortHeader !== sortHeader ||
        newSearchQuery !== searchQuery
      ) {
        newQueryParams.page = "0";
      }

      if (isRouteOk && teamIdForApi !== undefined) {
        newQueryParams.fleet_id = teamIdForApi;
      }

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_POLICIES,
        queryParams: { ...queryParams, ...newQueryParams },
      });

      if (isFirstNavigation.current) {
        isFirstNavigation.current = false;
        router?.replace(locationPath);
      } else {
        router?.push(locationPath);
      }
    },
    [
      isRouteOk,
      tableQueryDataForApi,
      sortDirection,
      sortHeader,
      searchQuery,
      teamIdForApi,
      queryParams,
      router,
    ] // Other dependencies can cause infinite re-renders as URL is source of truth
  );

  const toggleDeletePoliciesModal = () =>
    setShowDeletePoliciesModal(!showDeletePoliciesModal);

  const toggleAutomationsModal = () =>
    setShowAutomationsModal(!showAutomationsModal);

  const onOpenManageAutomationsModal = (policy: IPolicyStats) =>
    setSelectedPolicyForAutomations(policy);

  const onCloseManageAutomationsModal = () =>
    setSelectedPolicyForAutomations(null);

  const onAddPolicyClick = () => {
    setLastEditedQueryName("");
    setLastEditedQueryDescription("");
    setLastEditedQueryResolution("");
    setLastEditedQueryCritical(false);
    setLastEditedQueryHidden(false);
    setPolicyTeamId(
      currentTeamId === API_ALL_TEAMS_ID
        ? APP_CONTEXT_ALL_TEAMS_ID
        : currentTeamId
    );
    setLastEditedQueryBody(DEFAULT_POLICY.query);
    setLastEditedQueryId(null);
    router.push(
      currentTeamId === API_ALL_TEAMS_ID
        ? PATHS.NEW_POLICY
        : `${PATHS.NEW_POLICY}?fleet_id=${currentTeamId}`
    );
  };

  const onDeletePoliciesClick = (selectedTableIds: number[]): void => {
    toggleDeletePoliciesModal();
    setSelectedPolicyIds(selectedTableIds);
  };

  const onDeletePolicySubmit = useCallback(async () => {
    setIsUpdatingPolicies(true);
    try {
      // One request deletes policies from any scope, including the inherited
      // policies Primo mode lists in a fleet view.
      await policiesAPI.destroy(selectedPolicyIds);
      notify.success("Successfully deleted policies.");
    } catch (e) {
      // Deletes across scopes aren't atomic, so some may be gone; refresh
      // rather than retry the same IDs.
      notify.error(
        "Unable to delete policies. Some may have been deleted. Please try again.",
        { response: e }
      );
    } finally {
      setResetSelectedRows(true);
      setSelectedPolicyIds([]);
      invalidateAllPolicies();
      toggleDeletePoliciesModal();
      setIsUpdatingPolicies(false);
    }
  }, [
    invalidateAllPolicies,
    selectedPolicyIds,
    setResetSelectedRows,
    toggleDeletePoliciesModal,
  ]);

  const onChangeAutomationFilter = (val: SingleValue<CustomOptionType>) => {
    const automationType = val?.value;

    const locationPath = getNextLocationPath({
      pathPrefix: PATHS.MANAGE_POLICIES,
      queryParams: {
        ...queryParams,
        page: "0",
        automation_type: automationType === "all" ? undefined : automationType,
      },
    });

    router?.push(locationPath);
  };

  const policiesErrors = !isAllTeamsSelected
    ? teamPoliciesError
    : globalPoliciesError;

  const policyResults = !isAllTeamsSelected
    ? teamPolicies !== undefined
    : globalPolicies !== undefined;

  // Show CTA buttons if there are no errors
  const showCtaButtons = !policiesErrors;

  // A listed row of this fleet's own also counts, so a failed count request
  // doesn't hide the fleet's policies from automations and delete.
  const hasPoliciesToAutomate = isAllTeamsSelected
    ? (globalPoliciesCount ?? 0) > 0
    : (teamOwnPoliciesCount ?? 0) > 0 ||
      !!teamPolicies?.some((policy) => policy.fleet_id === teamIdForApi);
  const hasPoliciesToDelete =
    hasPoliciesToAutomate || (isPrimoMode && (teamPolicies?.length ?? 0) > 0); // in Primo mode, allow deleting inherited policies, which will be included in teamPolicies, from this view

  // Open the Manage automations modal via deep-link (e.g. from the
  // command palette). Gate on the same predicate the in-page button
  // uses — the param alone must not surface a privileged modal to
  // non-admins or when there's nothing to automate. Wait for the
  // relevant count query to settle (data OR error) so
  // `hasPoliciesToAutomate` is meaningful; then always strip the param
  // so a refresh doesn't reopen and the URL doesn't get stuck if the
  // count call errored.
  useEffect(() => {
    if (location.query.manage_automations !== "1") return;
    const countSettled = isAllTeamsSelected
      ? globalPoliciesCount !== undefined || isErrorGlobalPoliciesCount
      : teamOwnPoliciesCount !== undefined || isErrorTeamOwnPoliciesCount;
    if (!countSettled) return;
    if (canEditAutomationsSettings && hasPoliciesToAutomate) {
      setShowAutomationsModal(true);
    }
    const rest = { ...location.query };
    delete rest.manage_automations;
    router.replace({ pathname: location.pathname, query: rest });
  }, [
    location.query,
    location.pathname,
    router,
    canEditAutomationsSettings,
    hasPoliciesToAutomate,
    isAllTeamsSelected,
    globalPoliciesCount,
    teamOwnPoliciesCount,
    isErrorGlobalPoliciesCount,
    isErrorTeamOwnPoliciesCount,
  ]);

  const toOtherAutomationType = (
    config: IConfig | ITeamConfig | undefined
  ): OtherAutomationType | undefined => {
    const state = getTicketOrWebhookState(config);
    return state === "disabled" ? undefined : state;
  };
  const otherAutomationType = toOtherAutomationType(automationsConfig);
  // Inherited policies use the "All fleets" webhook or ticket destination.
  const inheritedOtherAutomationType = toOtherAutomationType(globalConfig);

  const renderPoliciesCountAndLastUpdated = (
    count?: number,
    policies?: IPolicyStats[]
  ) => {
    // Hide count if fetching count || there are errors OR there are no policy results with no filters (search or automation dropdown)
    const isFetchingCount = !isAllTeamsSelected
      ? isFetchingTeamCountMergeInherited
      : isFetchingGlobalCount;

    const hide =
      isFetchingCount ||
      policiesErrors ||
      (!policyResults &&
        searchQuery === "" &&
        !automationFilter &&
        !targetedPlatformParam);

    if (hide) {
      return null;
    }
    // Figure the time since the host counts were updated by finding first policy item with host_count_updated_at.
    const updatedAt =
      policies?.find((p) => !!p.host_count_updated_at)?.host_count_updated_at ||
      "";

    return (
      <>
        <TableCount name="policies" count={count} />
        <LastUpdatedText
          lastUpdatedAt={updatedAt}
          customTooltipText={
            <>
              Counts are updated hourly. Click host counts for the most
              up-to-date count.
            </>
          }
        />
      </>
    );
  };

  const automationFilterOptions: CustomOptionType[] = [
    {
      label: "All automations",
      value: "all",
      helpText: "All policies added to Fleet.",
    },
    {
      label: "Software",
      value: "software",
      helpText: "Policies with software automation enabled.",
    },
    {
      label: "Patch",
      value: "patch",
      helpText: "Patch policies for Fleet-maintained apps.",
    },
    {
      label: "Scripts",
      value: "script",
      helpText: "Policies with script automation enabled.",
    },
    {
      label: "Profiles",
      value: "configuration_profile",
      helpText: "Policies with configuration profile automation enabled.",
    },
    {
      label: "Calendar",
      value: "calendar",
      helpText: "Policies with calendar event automation enabled.",
    },
    {
      label: "Conditional access",
      value: "conditional_access",
      helpText: "Policies with conditional access automation enabled.",
    },
    {
      label: "Webhooks or tickets",
      value: "webhook_or_ticket",
      helpText: "Policies with webhook or ticket automation enabled.",
    },
  ];

  const allPoliciesOption = automationFilterOptions[0]; // value: "all"

  const getSelectedFilterOption = () => {
    if (!automationFilter) {
      return allPoliciesOption; // Default to all policies option
    }
    return automationFilterOptions.find(
      (opt) => opt.value === automationFilter
    );
  };

  const renderAutomationFilter = isPremiumTier
    ? () => {
        // Hide dropdown only on errors
        if (policiesErrors) {
          return null;
        }

        const policiesCount = isAllTeamsSelected
          ? globalPoliciesCount
          : teamPoliciesCountMergeInherited;
        const isTrulyEmpty =
          (policiesCount ?? 0) === 0 &&
          searchQuery === "" &&
          !automationFilter &&
          !targetedPlatformParam;

        const validAutomationTypesForTeam = getValidAutomationTypesForTeam(
          teamIdForApi
        );
        const optionsForTeam = automationFilterOptions.filter(
          (opt) =>
            opt.value === "all" ||
            (validAutomationTypesForTeam as string[]).includes(opt.value)
        );

        return (
          <DropdownWrapper
            className={`${baseClass}__filter-automation-dropdown`}
            name="filter-by-automation"
            value={getSelectedFilterOption()}
            onChange={onChangeAutomationFilter}
            placeholder="Filter by automation"
            options={optionsForTeam}
            variant="table-filter"
            isDisabled={isTrulyEmpty}
          />
        );
      }
    : undefined;

  const renderMainTable = () => {
    if (!isRouteOk || (isPremiumTier && !userTeams)) {
      return <Spinner />;
    }
    if (isAllTeamsSelected) {
      // Global policies

      if (globalPoliciesError) {
        return <TableDataError verticalPaddingSize="pad-xxxlarge" />;
      }
      return (
        <PoliciesTable
          policiesList={globalPolicies || []}
          isLoading={isFetchingGlobalPolicies || isFetchingGlobalConfig}
          onDeletePoliciesClick={onDeletePoliciesClick}
          onAddPolicyClick={onAddPolicyClick}
          canAddOrDeletePolicies={canAddOrDeletePolicies}
          hasPoliciesToDelete={hasPoliciesToDelete}
          currentTeam={currentTeamSummary}
          isPremiumTier={isPremiumTier}
          renderPoliciesCount={() =>
            renderPoliciesCountAndLastUpdated(
              globalPoliciesCount,
              globalPolicies
            )
          }
          count={globalPoliciesCount || 0}
          searchQuery={searchQuery}
          sortHeader={sortHeader}
          sortDirection={sortDirection}
          page={page}
          onQueryChange={onQueryChange}
          customControl={renderAutomationFilter}
          isFiltered={!!automationFilter || !!targetedPlatformParam}
          router={router}
          queryParams={queryParams}
          platform={targetedPlatformParam}
          otherAutomationType={otherAutomationType}
          inheritedOtherAutomationType={inheritedOtherAutomationType}
          onOpenManageAutomationsModal={
            canAddOrDeletePolicies ? onOpenManageAutomationsModal : undefined
          }
        />
      );
    }

    // Team policies
    if (teamPoliciesError) {
      return <TableDataError verticalPaddingSize="pad-xxxlarge" />;
    }
    const displayedTeamPolicies = teamPolicies || [];

    return (
      <div>
        <PoliciesTable
          policiesList={displayedTeamPolicies}
          isLoading={
            isFetchingTeamPolicies ||
            isFetchingTeamConfig ||
            isFetchingGlobalConfig
          }
          onDeletePoliciesClick={onDeletePoliciesClick}
          onAddPolicyClick={onAddPolicyClick}
          canAddOrDeletePolicies={canAddOrDeletePolicies}
          hasPoliciesToDelete={hasPoliciesToDelete}
          currentTeam={currentTeamSummary}
          renderPoliciesCount={() =>
            renderPoliciesCountAndLastUpdated(
              teamPoliciesCountMergeInherited,
              displayedTeamPolicies
            )
          }
          isPremiumTier={isPremiumTier}
          count={teamPoliciesCountMergeInherited || 0}
          searchQuery={searchQuery}
          sortHeader={sortHeader}
          sortDirection={sortDirection}
          page={page}
          onQueryChange={onQueryChange}
          customControl={renderAutomationFilter}
          isFiltered={!!automationFilter || !!targetedPlatformParam}
          router={router}
          queryParams={queryParams}
          platform={targetedPlatformParam}
          otherAutomationType={otherAutomationType}
          inheritedOtherAutomationType={inheritedOtherAutomationType}
          onOpenManageAutomationsModal={
            canAddOrDeletePolicies ? onOpenManageAutomationsModal : undefined
          }
        />
      </div>
    );
  };

  let automationsButton = null;
  if (canEditAutomationsSettings) {
    automationsButton = (
      <AutomationsButton
        onClick={toggleAutomationsModal}
        disabled={!hasPoliciesToAutomate}
      />
    );
    if (!hasPoliciesToAutomate) {
      const tipContent =
        isPremiumTier &&
        currentTeamId !== APP_CONTEXT_ALL_TEAMS_ID &&
        !globalConfigFromContext?.partnerships?.enable_primo ? (
          <div className={`${baseClass}__header__tooltip`}>
            To manage automations add a policy to this fleet.
            <br />
            For inherited policies select &ldquo;All fleets&rdquo;.
          </div>
        ) : (
          <div className={`${baseClass}__header__tooltip`}>
            To manage automations add a policy.
          </div>
        );

      automationsButton = (
        <TooltipWrapper
          underline={false}
          tipContent={tipContent}
          position="top"
          showArrow
        >
          {automationsButton}
        </TooltipWrapper>
      );
    }
  }

  if (!isRouteOk) {
    return <Spinner />;
  }

  const renderHeader = () => {
    if (isPremiumTier && !isPrimoMode) {
      if ((userTeams && userTeams.length > 1) || isOnGlobalTeam) {
        return (
          <FleetsDropdown
            currentUserFleets={userTeams || []}
            selectedFleetId={currentTeamId}
            onChange={onTeamChange}
            includeUnassigned
          />
        );
      }
      if (!isOnGlobalTeam && userTeams && userTeams.length === 1) {
        return <h1>{userTeams[0].name}</h1>;
      }
    }

    return <h1>Policies</h1>;
  };

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>{renderHeader()}</div>
            </div>

            {showCtaButtons && (
              <div className={`${baseClass} button-wrap`}>
                {automationsButton}
                {canAddOrDeletePolicies && (
                  <div className={`${baseClass}__action-button-container`}>
                    <Button
                      className={`${baseClass}__select-policy-button`}
                      onClick={onAddPolicyClick}
                    >
                      Add policy
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>
          <PageDescription content={"Detect device health issues."} />
        </div>
        {renderMainTable()}
        {showDeletePoliciesModal && (
          <DeletePoliciesModal
            isUpdatingPolicies={isUpdatingPolicies}
            onCancel={toggleDeletePoliciesModal}
            onSubmit={onDeletePolicySubmit}
          />
        )}
        {showAutomationsModal && (
          <AutomationsModal
            router={router}
            isAllTeamsSelected={isAllTeamsSelected}
            teamIdForApi={teamIdForApi}
            globalConfig={globalConfig}
            teamConfig={teamConfig}
            gitOpsModeEnabled={
              globalConfig?.gitops.gitops_mode_enabled ?? false
            }
            refetchPolicies={() => refetchPolicies(teamIdForApi)}
            onExit={toggleAutomationsModal}
          />
        )}
        {selectedPolicyForAutomations &&
          (() => {
            // An inherited policy (fleet_id === null) is global even when viewed
            // from within a fleet's list — its automations live on the global
            // config, so route the modal there.
            const isInheritedGlobal =
              selectedPolicyForAutomations.fleet_id === null;
            const modalAutomationsConfig = isInheritedGlobal
              ? globalConfig
              : automationsConfig;
            return (
              <ManageAutomationsModal
                policy={selectedPolicyForAutomations}
                fleetName={
                  isInheritedGlobal
                    ? "All fleets"
                    : currentTeamSummary?.name ?? ""
                }
                isGlobalPolicy={isInheritedGlobal}
                teamIdForApi={teamIdForApi}
                automationsConfig={modalAutomationsConfig}
                globalConfig={globalConfig}
                refetchPolicies={() => refetchPolicies(teamIdForApi)}
                onExit={onCloseManageAutomationsModal}
              />
            );
          })()}
      </>
    </MainContent>
  );
};

export default ManagePolicyPage;
