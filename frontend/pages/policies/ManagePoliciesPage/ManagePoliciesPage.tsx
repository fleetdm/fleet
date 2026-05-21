// TODO: make 'queryParams', 'router', and 'tableQueryData' dependencies stable (aka, memoized)
import React, {
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import PATHS from "router/paths";
import { isEqual } from "lodash";

import { getNextLocationPath } from "utilities/helpers";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IConfig } from "interfaces/config";
import {
  IPolicyStats,
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
  IPoliciesCountResponse,
  OtherAutomationType,
} from "interfaces/policy";
import { API_ALL_TEAMS_ID, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";

import configAPI from "services/entities/config";
import globalPoliciesAPI, {
  GlobalPoliciesAutomationType,
  IPoliciesCountQueryKey,
  IPoliciesQueryKey,
} from "services/entities/global_policies";
import teamPoliciesAPI, {
  ITeamPoliciesCountQueryKey,
  ITeamPoliciesQueryKey,
  AutomationType,
} from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import LastUpdatedText from "components/LastUpdatedText";
import TooltipWrapper from "components/TooltipWrapper";

import PoliciesTable from "./components/PoliciesTable";
import DeletePoliciesModal from "./components/DeletePoliciesModal";
import { DEFAULT_POLICY } from "../constants";
import AutomationsModal from "./components/AutomationsModal";
import ManageAutomationsModal from "./components/ManageAutomationsModal";

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
      automation_type?: AutomationType;
    };
    search: string;
  };
}

export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 20;
export const DEFAULT_SORT_COLUMN = "name";

const AUTOMATION_TYPES: AutomationType[] = [
  "software",
  "scripts",
  "calendar",
  "conditional_access",
  "other",
];

const GLOBAL_AUTOMATION_TYPES: GlobalPoliciesAutomationType[] = ["other"];

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

  const { renderFlash } = useContext(NotificationContext);
  const { setResetSelectedRows } = useContext(TableContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
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
  const initialAutomationFilter = (() => {
    const automationQueryParam = queryParams.automation_type;

    if (!automationQueryParam) {
      return null;
    }

    const validValues = isAllTeamsSelected
      ? GLOBAL_AUTOMATION_TYPES
      : AUTOMATION_TYPES;

    return (validValues as string[]).includes(automationQueryParam)
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
  const [automationFilter, setAutomationFilter] = useState<
    AutomationType | GlobalPoliciesAutomationType | null
  >(initialAutomationFilter);

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
        scope: "globalPolicies",
        page,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        automationType: automationFilter as GlobalPoliciesAutomationType,
      },
    ],
    ({ queryKey }) => {
      return globalPoliciesAPI.loadAllNew(queryKey[0]);
    },
    {
      enabled: isRouteOk && isAllTeamsSelected,
      select: (data) => data.policies || [],
      staleTime: 5000,
    }
  );

  const {
    data: globalPoliciesCount,
    isFetching: isFetchingGlobalCount,
    refetch: refetchGlobalPoliciesCount,
  } = useQuery<IPoliciesCountResponse, Error, number, IPoliciesCountQueryKey[]>(
    [
      {
        scope: "policiesCount",
        query: !isAllTeamsSelected ? "" : searchQuery,
        automationType: automationFilter as GlobalPoliciesAutomationType,
      },
    ],
    ({ queryKey }) => globalPoliciesAPI.getCount(queryKey[0]),
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
    ILoadTeamPoliciesResponse,
    Error,
    IPolicyStats[],
    ITeamPoliciesQueryKey[]
  >(
    [
      {
        scope: "teamPolicies",
        page,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        // teamIdForApi will never actually be undefined here
        teamId: teamIdForApi || 0,
        // no teams does inherit
        mergeInherited: true,
        automationType: automationFilter as AutomationType,
      },
    ],
    ({ queryKey }) => {
      return teamPoliciesAPI.loadAllNew(queryKey[0]);
    },
    {
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
      select: (data: ILoadTeamPoliciesResponse) => data.policies || [],
    }
  );

  const {
    data: teamPoliciesCountResponse,
    isFetching: isFetchingTeamCountMergeInherited,
    refetch: refetchTeamPoliciesCountMergeInherited,
  } = useQuery<
    IPoliciesCountResponse,
    Error,
    IPoliciesCountResponse,
    ITeamPoliciesCountQueryKey[]
  >(
    [
      {
        scope: "teamPoliciesCountMergeInherited",
        query: searchQuery,
        teamId: teamIdForApi || 0, // TODO: Fix number/undefined type
        mergeInherited: true,
        automationType: automationFilter as AutomationType,
      },
    ],
    ({ queryKey }) => teamPoliciesAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
    }
  );

  const teamPoliciesCountMergeInherited = teamPoliciesCountResponse?.count;

  const canAddOrDeletePolicies =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations = canAddOrDeletePolicies;

  const { data: globalConfig, isFetching: isFetchingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(
    ["config"],
    () => {
      return configAPI.loadAll();
    },
    {
      enabled: isRouteOk && canAddOrDeletePolicies,
      onSuccess: (data) => {
        setConfig(data);
      },
      staleTime: 5000,
    }
  );

  const { data: teamData, isFetching: isFetchingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error
  >(["teams", teamIdForApi], () => teamsAPI.load(teamIdForApi), {
    // Enable for all teams including "No team" (teamIdForApi === 0)
    enabled: isRouteOk && teamIdForApi !== undefined && canAddOrDeletePolicies,
    staleTime: 5000,
  });
  const teamConfig = teamData?.team;

  const automationsConfig = isAllTeamsSelected ? globalConfig : teamConfig;

  const refetchPolicies = (teamId?: number) => {
    if (teamId !== undefined) {
      refetchTeamPolicies();
      refetchTeamPoliciesCountMergeInherited();
    } else {
      refetchGlobalPolicies(); // Only call on global policies as this is expensive
      refetchGlobalPoliciesCount();
    }
  };

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
      const responses: Promise<any>[] = [];
      if (isPrimoMode) {
        // filter selected policies by All team and no team
        const selectedSet = new Set(selectedPolicyIds); // more efficient for below reduce
        const [
          globalPolicyIdsToDelete,
          teamPolicyIdsToDelete, // will be No team, since this is Primo mode
        ] = (teamPolicies ?? []).reduce(
          (acc, policy) => {
            if (selectedSet.has(policy.id)) {
              // need to compare policy team id
              if (policy.team_id === null) {
                // note `null` not `undefined` here
                acc[0].push(policy.id);
              } else {
                acc[1].push(policy.id);
              }
            }
            return acc;
          },
          [[], []] as [number[], number[]]
        );
        // delete all team policies via global endpoint, No team via team endpoint
        if (globalPolicyIdsToDelete.length) {
          responses.push(globalPoliciesAPI.destroy(globalPolicyIdsToDelete));
        }
        if (teamPolicyIdsToDelete.length) {
          responses.push(
            teamPoliciesAPI.destroy(teamIdForApi, teamPolicyIdsToDelete)
          );
        }
      } else {
        // normal Fleet operation
        responses.push(
          !isAllTeamsSelected
            ? teamPoliciesAPI.destroy(teamIdForApi, selectedPolicyIds)
            : globalPoliciesAPI.destroy(selectedPolicyIds)
        );
      }

      await Promise.all(responses);
      renderFlash("success", "Successfully deleted policies.");
      setResetSelectedRows(true);
      refetchPolicies(teamIdForApi);
    } catch {
      renderFlash("error", "Unable to delete policies. Please try again.");
    } finally {
      toggleDeletePoliciesModal();
      setIsUpdatingPolicies(false);
    }
  }, [
    isAllTeamsSelected,
    isPrimoMode,
    refetchPolicies,
    renderFlash,
    selectedPolicyIds,
    setResetSelectedRows,
    teamIdForApi,
    teamPolicies,
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

  const hasPoliciesToAutomate = isAllTeamsSelected
    ? (globalPoliciesCount ?? 0) > 0
    : (teamPoliciesCountMergeInherited ?? 0) >
      (teamPoliciesCountResponse?.inherited_policy_count ?? 0);
  const hasPoliciesToDelete =
    hasPoliciesToAutomate || (isPrimoMode && (teamPolicies?.length ?? 0) > 0); // in Primo mode, allow deleting inherited policies, which will be included in teamPolicies, from this view

  // NOTE: backend uses webhook_settings to store automated policy ids for both webhooks and integrations
  let currentAutomatedPolicies: number[] = [];
  let otherAutomationType: OtherAutomationType | undefined;
  if (automationsConfig) {
    const {
      webhook_settings: { failing_policies_webhook: webhook },
      integrations,
    } = automationsConfig;

    let isIntegrationEnabled = false;
    if (integrations) {
      const { jira, zendesk } = integrations;
      isIntegrationEnabled =
        !!jira?.find((j) => j.enable_failing_policies) ||
        !!zendesk?.find((z) => z.enable_failing_policies);
    }

    if (isIntegrationEnabled || webhook?.enable_failing_policies_webhook) {
      currentAutomatedPolicies = webhook?.policy_ids || [];
    }

    if (isIntegrationEnabled) {
      otherAutomationType = "ticket";
    } else if (webhook?.enable_failing_policies_webhook) {
      otherAutomationType = "webhook";
    }
  }

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
      (!policyResults && searchQuery === "" && !automationFilter);

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
              Counts are updated hourly. Click host
              <br />
              counts for the most up-to-date count.
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
      label: "Scripts",
      value: "scripts",
      helpText: "Policies with script automation enabled.",
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
      value: "other",
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
          (policiesCount ?? 0) === 0 && searchQuery === "" && !automationFilter;

        // No team ID = All fleets → only show "all" and "other" options
        const optionsForTeam = teamIdForApi
          ? automationFilterOptions
          : automationFilterOptions.filter((opt) =>
              ["all", "other"].includes(opt.value as string)
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
          currentAutomatedPolicies={currentAutomatedPolicies}
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
          isFiltered={!!automationFilter}
          otherAutomationType={otherAutomationType}
          onOpenManageAutomationsModal={
            canManageAutomations ? onOpenManageAutomationsModal : undefined
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
          currentAutomatedPolicies={currentAutomatedPolicies}
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
          isFiltered={!!automationFilter}
          otherAutomationType={otherAutomationType}
          onOpenManageAutomationsModal={
            canManageAutomations ? onOpenManageAutomationsModal : undefined
          }
        />
      </div>
    );
  };

  let automationsButton = null;
  if (canManageAutomations) {
    automationsButton = (
      <Button
        className={`${baseClass}__automations-button`}
        onClick={toggleAutomationsModal}
        disabled={!hasPoliciesToAutomate}
        variant="inverse"
      >
        <>
          <Icon name="settings" /> Automations
        </>
      </Button>
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
          <TeamsDropdown
            currentUserTeams={userTeams || []}
            selectedTeamId={currentTeamId}
            onChange={onTeamChange}
            includeNoTeams
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
        {selectedPolicyForAutomations && (
          <ManageAutomationsModal
            policy={selectedPolicyForAutomations}
            onExit={onCloseManageAutomationsModal}
          />
        )}
      </>
    </MainContent>
  );
};

export default ManagePolicyPage;
