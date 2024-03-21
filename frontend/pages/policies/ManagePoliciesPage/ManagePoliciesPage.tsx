import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import PATHS from "router/paths";
import { noop, isEqual } from "lodash";

import { getNextLocationPath } from "utilities/helpers";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IConfig, IWebhookSettings } from "interfaces/config";
import { IIntegrations } from "interfaces/integration";
import {
  IPolicyStats,
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
  IPoliciesCountResponse,
} from "interfaces/policy";
import { ITeamConfig } from "interfaces/team";

import configAPI from "services/entities/config";
import globalPoliciesAPI, {
  IPoliciesCountQueryKey,
  IPoliciesQueryKey,
} from "services/entities/global_policies";
import teamPoliciesAPI, {
  ITeamPoliciesCountQueryKey,
  ITeamPoliciesQueryKey,
} from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import { ITableQueryData } from "components/TableContainer/TableContainer";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";

import PoliciesTable from "./components/PoliciesTable";
import ManagePolicyAutomationsModal from "./components/ManagePolicyAutomationsModal";
import AddPolicyModal from "./components/AddPolicyModal";
import DeletePolicyModal from "./components/DeletePolicyModal";

interface IManagePoliciesPageProps {
  router: InjectedRouter;
  location: {
    action: string;
    hash: string;
    key: string;
    pathname: string;
    query: {
      team_id?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      page?: string;
      inherited_table?: "true";
      inherited_order_key?: string;
      inherited_order_direction?: "asc" | "desc";
      inherited_page?: string;
    };
    search: string;
  };
}

const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_SORT_COLUMN = "name";

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
    isFreeTier,
    isPremiumTier,
    isSandboxMode,
    setConfig,
    setFilteredPoliciesPath,
    filteredPoliciesPath,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const { setResetSelectedRows } = useContext(TableContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const {
    currentTeamId,
    currentTeamName,
    currentTeamSummary,
    isAnyTeamSelected,
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
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: true,
      observer_plus: true,
    },
  });

  const [isUpdatingAutomations, setIsUpdatingAutomations] = useState(false);
  const [isUpdatingPolicies, setIsUpdatingPolicies] = useState(false);
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showDeletePolicyModal, setShowDeletePolicyModal] = useState(false);

  const [teamPolicies, setTeamPolicies] = useState<IPolicyStats[]>();
  const [inheritedPolicies, setInheritedPolicies] = useState<IPolicyStats[]>();

  // Functions to avoid race conditions
  const initialSearchQuery = (() => queryParams.query ?? "")();
  const initialSortHeader = (() =>
    (queryParams?.order_key as "name" | "failing_host_count") ??
    DEFAULT_SORT_COLUMN)();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ??
    DEFAULT_SORT_DIRECTION)();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();
  const initialShowInheritedTable = (() =>
    queryParams && queryParams.inherited_table === "true")();
  const initialInheritedSortHeader = (() =>
    (queryParams?.inherited_order_key as "name" | "failing_host_count") ??
    DEFAULT_SORT_COLUMN)();
  const initialInheritedSortDirection = (() =>
    (queryParams?.inherited_order_direction as "asc" | "desc") ??
    DEFAULT_SORT_DIRECTION)();
  const initialInheritedPage = (() =>
    queryParams && queryParams.inherited_page
      ? parseInt(queryParams?.inherited_page, 10)
      : 0)();

  const showInheritedTable = initialShowInheritedTable;

  // Needs update on location change or table state might not match URL
  const [searchQuery, setSearchQuery] = useState(initialSearchQuery);
  const [page, setPage] = useState(initialPage);
  const [inheritedPage, setInheritedPage] = useState(initialInheritedPage);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [
    inheritedTableQueryData,
    setInheritedTableQueryData,
  ] = useState<ITableQueryData>();
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(initialSortDirection);
  const [inheritedSortDirection, setInheritedSortDirection] = useState(
    initialInheritedSortDirection
  );
  const [inheritedSortHeader, setInheritedSortHeader] = useState(
    initialInheritedSortHeader
  );

  useEffect(() => {
    setLastEditedQueryPlatform(null);
  }, []);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    setPage(initialPage);
    setSearchQuery(initialSearchQuery);
    setSortHeader(initialSortHeader);
    setSortDirection(initialSortDirection);
    setInheritedPage(initialInheritedPage);
    setInheritedSortHeader(initialInheritedSortHeader);
    setInheritedSortDirection(initialInheritedSortDirection);
  }, [location, isRouteOk]);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    const path = location.pathname + location.search;
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
        page: tableQueryData?.pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
      },
    ],
    ({ queryKey }) => {
      return globalPoliciesAPI.loadAllNew(queryKey[0]);
    },
    {
      enabled: isRouteOk && !isAnyTeamSelected,
      select: (data) => data.policies,
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
        query: isAnyTeamSelected ? "" : searchQuery, // Search query not used for inherited count
      },
    ],
    ({ queryKey }) => globalPoliciesAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  const {
    error: teamPoliciesError,
    isFetching: isFetchingTeamPolicies,
    refetch: refetchTeamPolicies,
  } = useQuery<
    ILoadTeamPoliciesResponse,
    Error,
    ILoadTeamPoliciesResponse,
    ITeamPoliciesQueryKey[]
  >(
    [
      {
        scope: "teamPolicies",
        page: tableQueryData?.pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        inheritedPage: inheritedTableQueryData?.pageIndex,
        inheritedPerPage: DEFAULT_PAGE_SIZE,
        inheritedOrderDirection: inheritedSortDirection,
        inheritedOrderKey: inheritedSortHeader,
        teamId: teamIdForApi || 0,
      },
    ],
    ({ queryKey }) => {
      return teamPoliciesAPI.loadAllNew(queryKey[0]);
    },
    {
      enabled: isRouteOk && isPremiumTier && !!teamIdForApi,
      onSuccess: (data) => {
        setTeamPolicies(data.policies);
        setInheritedPolicies(data.inherited_policies);
      },
    }
  );

  const {
    data: teamPoliciesCount,
    isFetching: isFetchingTeamCount,
    refetch: refetchTeamPoliciesCount,
  } = useQuery<
    IPoliciesCountResponse,
    Error,
    number,
    ITeamPoliciesCountQueryKey[]
  >(
    [
      {
        scope: "teamPoliciesCount",
        query: searchQuery,
        teamId: teamIdForApi || 0, // TODO: Fix number/undefined type
      },
    ],
    ({ queryKey }) => teamPoliciesAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && !!teamIdForApi,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  const canAddOrDeletePolicy: boolean =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations: boolean = isGlobalAdmin || isTeamAdmin;

  const {
    data: config,
    isFetching: isFetchingConfig,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error>(
    ["config"],
    () => {
      return configAPI.loadAll();
    },
    {
      enabled: canAddOrDeletePolicy,
      onSuccess: (data) => {
        setConfig(data);
      },
      staleTime: 5000,
    }
  );

  const {
    data: teamConfig,
    isFetching: isFetchingTeamConfig,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teams", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi && canAddOrDeletePolicy,
      select: (data) => data.team,
    }
  );

  const refetchPolicies = (teamId?: number) => {
    if (teamId) {
      refetchTeamPolicies();
      refetchTeamPoliciesCount();
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
  // Inherited table uses the same onQueryChange function but routes to different URL params
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      if (!isRouteOk || isEqual(newTableQuery, tableQueryData)) {
        return;
      }

      newTableQuery.editingInheritedTable
        ? setInheritedTableQueryData({ ...newTableQuery })
        : setTableQueryData({ ...newTableQuery });

      const {
        pageIndex: newPageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
        sortHeader: newSortHeader,
        editingInheritedTable,
      } = newTableQuery;
      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};

      newQueryParams.query = newSearchQuery;

      // Updates main policy table URL params
      // No change to inherited policy table URL params
      if (!editingInheritedTable) {
        newQueryParams.order_key = newSortHeader;
        newQueryParams.order_direction = newSortDirection;
        newQueryParams.page = newPageIndex.toString();
        if (showInheritedTable) {
          newQueryParams.inherited_order_key = inheritedSortHeader;
          newQueryParams.inherited_order_direction = inheritedSortDirection;
          newQueryParams.inherited_page = inheritedPage.toString();
        }
        // Reset page number to 0 for new filters
        if (
          newSortDirection !== sortDirection ||
          newSortHeader !== sortHeader ||
          newSearchQuery !== searchQuery
        ) {
          newQueryParams.page = "0";
        }
      }

      if (showInheritedTable) {
        newQueryParams.inherited_table =
          showInheritedTable && showInheritedTable.toString();
      }

      // Updates inherited policy table URL params
      // No change to main policy table URL params
      if (showInheritedTable && editingInheritedTable) {
        newQueryParams.inherited_order_key = newSortHeader;
        newQueryParams.inherited_order_direction = newSortDirection;
        newQueryParams.inherited_page = newPageIndex.toString();
        newQueryParams.order_key = sortHeader;
        newQueryParams.order_direction = sortDirection;
        newQueryParams.page = page.toString();
        newQueryParams.query = searchQuery;
        // Reset page number to 0 for new filters
        if (
          newSortDirection !== inheritedSortDirection ||
          newSortHeader !== inheritedSortHeader
        ) {
          newQueryParams.inherited_page = "0";
        }
      }

      if (isRouteOk && teamIdForApi !== undefined) {
        newQueryParams.team_id = teamIdForApi;
      }

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_POLICIES,
        queryParams: { ...queryParams, ...newQueryParams },
      });

      router?.replace(locationPath);
    },
    [
      isRouteOk,
      teamIdForApi,
      searchQuery,
      showInheritedTable,
      inheritedSortDirection,
      sortDirection,
    ] // Other dependencies can cause infinite re-renders as URL is source of truth
  );

  const toggleManageAutomationsModal = () =>
    setShowManageAutomationsModal(!showManageAutomationsModal);

  const togglePreviewPayloadModal = useCallback(() => {
    setShowPreviewPayloadModal(!showPreviewPayloadModal);
  }, [setShowPreviewPayloadModal, showPreviewPayloadModal]);

  const toggleAddPolicyModal = () => setShowAddPolicyModal(!showAddPolicyModal);

  const toggleDeletePolicyModal = () =>
    setShowDeletePolicyModal(!showDeletePolicyModal);

  const toggleShowInheritedPolicies = () => {
    // URL source of truth
    const locationPath = getNextLocationPath({
      pathPrefix: PATHS.MANAGE_POLICIES,
      queryParams: {
        ...queryParams,
        inherited_table: showInheritedTable ? undefined : "true",
        inherited_page: showInheritedTable ? undefined : "0",
      },
    });
    router?.replace(locationPath);
  };

  const handleUpdateAutomations = async (requestBody: {
    webhook_settings: Pick<IWebhookSettings, "failing_policies_webhook">;
    integrations: IIntegrations;
  }) => {
    setIsUpdatingAutomations(true);
    try {
      await (isAnyTeamSelected
        ? teamsAPI.update(requestBody, teamIdForApi)
        : configAPI.update(requestBody));
      renderFlash("success", "Successfully updated policy automations.");
    } catch {
      renderFlash(
        "error",
        "Could not update policy automations. Please try again."
      );
    } finally {
      toggleManageAutomationsModal();
      setIsUpdatingAutomations(false);
      refetchConfig();
      isAnyTeamSelected && refetchTeamConfig();
    }
  };

  const onAddPolicyClick = () => {
    setLastEditedQueryName("");
    setLastEditedQueryDescription("");
    setLastEditedQueryResolution("");
    setLastEditedQueryCritical(false);
    toggleAddPolicyModal();
  };

  const onDeletePolicyClick = (selectedTableIds: number[]): void => {
    toggleDeletePolicyModal();
    setSelectedPolicyIds(selectedTableIds);
  };

  const onDeletePolicySubmit = async () => {
    setIsUpdatingPolicies(true);
    try {
      const request = isAnyTeamSelected
        ? teamPoliciesAPI.destroy(teamIdForApi, selectedPolicyIds)
        : globalPoliciesAPI.destroy(selectedPolicyIds);

      await request.then(() => {
        renderFlash(
          "success",
          `Successfully deleted ${
            selectedPolicyIds?.length === 1 ? "policy" : "policies"
          }.`
        );
        setResetSelectedRows(true);
        refetchPolicies(teamIdForApi);
      });
    } catch {
      renderFlash(
        "error",
        `Unable to delete ${
          selectedPolicyIds?.length === 1 ? "policy" : "policies"
        }. Please try again.`
      );
    } finally {
      toggleDeletePolicyModal();
      setIsUpdatingPolicies(false);
    }
  };

  const inheritedPoliciesButtonText = (
    showPolicies: boolean,
    count: number
  ) => {
    return `${showPolicies ? "Hide" : "Show"} ${count} inherited ${
      count > 1 ? "policies" : "policy"
    }`;
  };

  const showInheritedPoliciesButton =
    isAnyTeamSelected &&
    !isFetchingTeamPolicies &&
    !teamPoliciesError &&
    !!inheritedPolicies?.length; // Returned with team policies

  const availablePoliciesForAutomation =
    (isAnyTeamSelected ? teamPolicies : globalPolicies) || [];

  const policiesErrors = isAnyTeamSelected
    ? teamPoliciesError
    : globalPoliciesError;

  const policyResults = isAnyTeamSelected ? !!teamPolicies : !!globalPolicies;

  // Show CTA buttons if there is no errors AND there are policy results or a search filter
  const showCtaButtons =
    !policiesErrors && (policyResults || searchQuery !== "");

  const automationsConfig = isAnyTeamSelected ? teamConfig : config;

  // NOTE: backend uses webhook_settings to store automated policy ids for both webhooks and integrations
  let currentAutomatedPolicies: number[] = [];
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
  }

  const renderPoliciesCount = (count?: number) => {
    // Show count if there is no errors AND there are policy results or a search filter
    const showCount =
      count !== undefined &&
      !policiesErrors &&
      (policyResults || searchQuery !== "");

    return (
      <div className={`${baseClass}__count`}>
        {showCount && (
          <span>{`${count} polic${count === 1 ? "y" : "ies"}`}</span>
        )}
      </div>
    );
  };

  const renderMainTable = () => {
    return !isRouteOk || (isPremiumTier && !userTeams) ? (
      <Spinner />
    ) : (
      <div>
        {isAnyTeamSelected && teamPoliciesError && <TableDataError />}
        {isAnyTeamSelected && !teamPoliciesError && (
          <PoliciesTable
            policiesList={teamPolicies || []}
            isLoading={
              isFetchingTeamPolicies || isFetchingTeamConfig || isFetchingConfig
            }
            onAddPolicyClick={onAddPolicyClick}
            onDeletePolicyClick={onDeletePolicyClick}
            canAddOrDeletePolicy={canAddOrDeletePolicy}
            currentTeam={currentTeamSummary}
            currentAutomatedPolicies={currentAutomatedPolicies}
            renderPoliciesCount={() =>
              !isFetchingTeamCount && renderPoliciesCount(teamPoliciesCount)
            }
            isPremiumTier={isPremiumTier}
            isSandboxMode={isSandboxMode}
            searchQuery={searchQuery}
            sortHeader={sortHeader}
            sortDirection={sortDirection}
            page={page}
            onQueryChange={onQueryChange}
          />
        )}
        {!isAnyTeamSelected && globalPoliciesError && <TableDataError />}
        {!isAnyTeamSelected && !globalPoliciesError && (
          <PoliciesTable
            policiesList={globalPolicies || []}
            isLoading={isFetchingGlobalPolicies || isFetchingConfig}
            onAddPolicyClick={onAddPolicyClick}
            onDeletePolicyClick={onDeletePolicyClick}
            canAddOrDeletePolicy={canAddOrDeletePolicy}
            currentTeam={currentTeamSummary}
            currentAutomatedPolicies={currentAutomatedPolicies}
            isPremiumTier={isPremiumTier}
            isSandboxMode={isSandboxMode}
            renderPoliciesCount={() =>
              !isFetchingGlobalCount && renderPoliciesCount(globalPoliciesCount)
            }
            searchQuery={searchQuery}
            sortHeader={sortHeader}
            sortDirection={sortDirection}
            page={page}
            onQueryChange={onQueryChange}
          />
        )}
      </div>
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Policies</h1>}
                {isPremiumTier &&
                  ((userTeams && userTeams.length > 1) || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={userTeams || []}
                      selectedTeamId={currentTeamId}
                      onChange={onTeamChange}
                      isSandboxMode={isSandboxMode}
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  userTeams &&
                  userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
              </div>
            </div>
          </div>
          {showCtaButtons && (
            <div className={`${baseClass} button-wrap`}>
              {canManageAutomations && automationsConfig && (
                <Button
                  onClick={toggleManageAutomationsModal}
                  className={`${baseClass}__manage-automations button`}
                  variant="inverse"
                  disabled={
                    isAnyTeamSelected
                      ? isFetchingTeamPolicies
                      : isFetchingGlobalPolicies
                  }
                >
                  <span>Manage automations</span>
                </Button>
              )}
              {canAddOrDeletePolicy && (
                <div className={`${baseClass}__action-button-container`}>
                  <Button
                    variant="brand"
                    className={`${baseClass}__select-policy-button`}
                    onClick={onAddPolicyClick}
                  >
                    Add a policy
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          <p>
            {isAnyTeamSelected
              ? "Detect device health issues for all hosts assigned to this team."
              : "Detect device health issues for all hosts."}
          </p>
        </div>
        {renderMainTable()}
        {showInheritedPoliciesButton && globalPoliciesCount && (
          <RevealButton
            isShowing={showInheritedTable}
            className={baseClass}
            hideText={inheritedPoliciesButtonText(
              showInheritedTable,
              globalPoliciesCount
            )}
            showText={inheritedPoliciesButtonText(
              showInheritedTable,
              globalPoliciesCount
            )}
            caretPosition="before"
            tooltipContent={
              <>
                &quot;All teams&quot; policies are checked
                <br />
                for this team&apos;s hosts.
              </>
            }
            onClick={toggleShowInheritedPolicies}
          />
        )}
        {showInheritedPoliciesButton && showInheritedTable && (
          <div className={`${baseClass}__inherited-policies-table`}>
            {globalPoliciesError && <TableDataError />}
            {!globalPoliciesError && (
              <PoliciesTable
                isLoading={isFetchingTeamPolicies}
                policiesList={inheritedPolicies || []}
                onDeletePolicyClick={noop}
                canAddOrDeletePolicy={canAddOrDeletePolicy}
                tableType="inheritedPolicies"
                currentTeam={currentTeamSummary}
                searchQuery=""
                renderPoliciesCount={() =>
                  renderPoliciesCount(teamPoliciesCount)
                }
                sortHeader={inheritedSortHeader}
                sortDirection={inheritedSortDirection}
                page={inheritedPage}
                onQueryChange={onQueryChange}
              />
            )}
          </div>
        )}
        {config && automationsConfig && showManageAutomationsModal && (
          <ManagePolicyAutomationsModal
            automationsConfig={automationsConfig}
            availableIntegrations={config.integrations}
            availablePolicies={availablePoliciesForAutomation}
            isUpdatingAutomations={isUpdatingAutomations}
            showPreviewPayloadModal={showPreviewPayloadModal}
            onExit={toggleManageAutomationsModal}
            handleSubmit={handleUpdateAutomations}
            togglePreviewPayloadModal={togglePreviewPayloadModal}
          />
        )}
        {showAddPolicyModal && (
          <AddPolicyModal
            onCancel={toggleAddPolicyModal}
            router={router}
            teamId={teamIdForApi || 0}
            teamName={currentTeamName}
          />
        )}
        {showDeletePolicyModal && (
          <DeletePolicyModal
            isUpdatingPolicies={isUpdatingPolicies}
            onCancel={toggleDeletePolicyModal}
            onSubmit={onDeletePolicySubmit}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManagePolicyPage;
