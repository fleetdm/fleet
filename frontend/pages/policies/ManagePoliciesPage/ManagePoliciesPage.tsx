import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import PATHS from "router/paths";
import { noop, isEmpty } from "lodash";

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
} from "interfaces/policy";
import { ITeamConfig } from "interfaces/team";

import configAPI from "services/entities/config";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import { ITableQueryData } from "components/TableContainer/TableContainer";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";

import PoliciesTable from "./components/PoliciesTable";
import ManageAutomationsModal from "./components/ManageAutomationsModal";
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
    (queryParams?.order_key as "name" | "failing_host_count") ?? "name")();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ?? "asc")();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();
  const initialShowInheritedTable = (() =>
    queryParams && queryParams.inherited_table === "true")();
  const initialInheritedSortHeader = (() =>
    (queryParams?.inherited_order_key as "name" | "failing_host_count") ??
    "name")();
  const initialInheritedSortDirection = (() =>
    (queryParams?.inherited_order_direction as "asc" | "desc") ?? "asc")();
  const initialInheritedPage = (() =>
    queryParams && queryParams.inherited_page
      ? parseInt(queryParams?.inherited_page, 10)
      : 0)();

  const page = initialPage;
  const showInheritedTable = initialShowInheritedTable;
  const inheritedPage = initialInheritedPage;
  const searchQuery = initialSearchQuery;

  // Needs update on location change or table state might not match URL
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  const [sortDirection, setSortDirection] = useState(initialSortDirection);
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
    setSortHeader(initialSortHeader);
    setSortDirection(initialSortDirection);
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
  } = useQuery<ILoadAllPoliciesResponse, Error, IPolicyStats[]>(
    ["globalPolicies", teamIdForApi],
    () => {
      return globalPoliciesAPI.loadAll();
    },
    {
      enabled: isRouteOk,
      select: (data) => data.policies,
      staleTime: 5000,
    }
  );

  const {
    error: teamPoliciesError,
    isFetching: isFetchingTeamPolicies,
    refetch: refetchTeamPolicies,
  } = useQuery<ILoadTeamPoliciesResponse, Error, ILoadTeamPoliciesResponse>(
    ["teamPolicies", teamIdForApi],
    () => teamPoliciesAPI.loadAll(teamIdForApi),
    {
      enabled: isRouteOk && isPremiumTier && !!teamIdForApi,
      onSuccess: (data) => {
        setTeamPolicies(data.policies);
        setInheritedPolicies(data.inherited_policies);
      },
    }
  );

  const canAddOrDeletePolicy =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations = isGlobalAdmin || isTeamAdmin;

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
    refetchGlobalPolicies();
    if (teamId) {
      refetchTeamPolicies();
    }
  };

  // const findAvailableTeam = (id: number) => {
  //   return availableTeams?.find((t) => t.id === id);
  // };

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

  const onClientSidePaginationChange = useCallback(
    (pageIndex: number) => {
      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_POLICIES,
        queryParams: {
          ...queryParams,
          page: pageIndex,
          query: searchQuery,
          order_direction: sortDirection,
          order_key: sortHeader,
          inherited_order_direction: inheritedSortDirection,
          inherited_order_key: inheritedSortHeader,
          inherited_page: inheritedPage,
        },
      });

      router?.replace(locationPath);
    },
    [searchQuery, queryParams, sortHeader, sortDirection] // Dependencies required for correct variable state
  );

  const onClientSideInheritedPaginationChange = useCallback(
    (pageIndex: number) => {
      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_POLICIES,
        queryParams: {
          ...queryParams,
          inherited_table: "true",
          inherited_page: pageIndex,
          query: searchQuery,
          page,
          order_direction: sortDirection,
          order_key: sortHeader,
          inherited_order_direction: inheritedSortDirection,
          inherited_order_key: inheritedSortHeader,
        },
      });
      router?.replace(locationPath);
    },
    [queryParams, inheritedSortHeader, inheritedSortDirection] // Dependencies required for correct variable state
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

  const showTeamDescription = isPremiumTier && isAnyTeamSelected;

  const showInheritedPoliciesButton =
    isAnyTeamSelected &&
    !isFetchingTeamPolicies &&
    !teamPoliciesError &&
    !isFetchingGlobalPolicies &&
    !globalPoliciesError &&
    !!globalPolicies?.length;

  const availablePoliciesForAutomation =
    (isAnyTeamSelected ? teamPolicies : globalPolicies) || [];

  const showCtaButtons =
    (isAnyTeamSelected && teamPolicies) ||
    (!isAnyTeamSelected && globalPolicies);

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

  return !isRouteOk || (isPremiumTier && !userTeams) ? (
    <Spinner />
  ) : (
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
              {canManageAutomations &&
                automationsConfig &&
                !isFetchingGlobalPolicies && (
                  <Button
                    onClick={toggleManageAutomationsModal}
                    className={`${baseClass}__manage-automations button`}
                    variant="inverse"
                  >
                    <span>Manage automations</span>
                  </Button>
                )}
              {canAddOrDeletePolicy &&
                ((isAnyTeamSelected && !isFetchingTeamPolicies) ||
                  !isFetchingGlobalPolicies) && (
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
          {showTeamDescription ? (
            <p>
              Add additional policies for <b>all hosts assigned to this team</b>
              .
            </p>
          ) : (
            <p>
              Add policies for <b>all of your hosts</b> to see which pass your
              organization’s standards.
            </p>
          )}
        </div>
        <div>
          {isAnyTeamSelected && teamPoliciesError && <TableDataError />}
          {isAnyTeamSelected &&
            !teamPoliciesError &&
            (isFetchingTeamPolicies ? (
              <Spinner />
            ) : (
              <PoliciesTable
                policiesList={teamPolicies || []}
                isLoading={
                  isFetchingTeamPolicies ||
                  isFetchingTeamConfig ||
                  isFetchingConfig
                }
                onAddPolicyClick={onAddPolicyClick}
                onDeletePolicyClick={onDeletePolicyClick}
                canAddOrDeletePolicy={canAddOrDeletePolicy}
                currentTeam={currentTeamSummary}
                currentAutomatedPolicies={currentAutomatedPolicies}
                isPremiumTier={isPremiumTier}
                isSandboxMode={isSandboxMode}
                searchQuery={searchQuery}
                sortHeader={sortHeader}
                sortDirection={sortDirection}
                page={page}
                onQueryChange={onQueryChange}
              />
            ))}
          {!isAnyTeamSelected && globalPoliciesError && <TableDataError />}
          {!isAnyTeamSelected &&
            !globalPoliciesError &&
            (isFetchingGlobalPolicies ? (
              <Spinner />
            ) : (
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
                onClientSidePaginationChange={onClientSidePaginationChange}
                searchQuery={searchQuery}
                sortHeader={sortHeader}
                sortDirection={sortDirection}
                page={page}
                onQueryChange={onQueryChange}
              />
            ))}
        </div>
        {showInheritedPoliciesButton && globalPolicies && (
          <RevealButton
            isShowing={showInheritedTable}
            className={baseClass}
            hideText={inheritedPoliciesButtonText(
              showInheritedTable,
              globalPolicies.length
            )}
            showText={inheritedPoliciesButtonText(
              showInheritedTable,
              globalPolicies.length
            )}
            caretPosition={"before"}
            tooltipHtml={
              '"All teams" policies are checked <br/> for this team’s hosts.'
            }
            onClick={toggleShowInheritedPolicies}
          />
        )}
        {showInheritedPoliciesButton && showInheritedTable && (
          <div className={`${baseClass}__inherited-policies-table`}>
            {globalPoliciesError && <TableDataError />}
            {!globalPoliciesError &&
              (isFetchingGlobalPolicies ? (
                <Spinner />
              ) : (
                <PoliciesTable
                  isLoading={isFetchingTeamPolicies}
                  policiesList={inheritedPolicies || []}
                  onDeletePolicyClick={noop}
                  canAddOrDeletePolicy={canAddOrDeletePolicy}
                  tableType="inheritedPolicies"
                  currentTeam={currentTeamSummary}
                  searchQuery=""
                  onClientSidePaginationChange={
                    onClientSideInheritedPaginationChange
                  }
                  sortHeader={inheritedSortHeader}
                  sortDirection={inheritedSortDirection}
                  page={inheritedPage}
                  onQueryChange={onQueryChange}
                />
              ))}
          </div>
        )}
        {config && automationsConfig && showManageAutomationsModal && (
          <ManageAutomationsModal
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
