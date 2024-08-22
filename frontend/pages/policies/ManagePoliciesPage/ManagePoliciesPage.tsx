import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import PATHS from "router/paths";
import { isEqual } from "lodash";

import { getNextLocationPath, wait } from "utilities/helpers";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IConfig, IWebhookSettings } from "interfaces/config";
import { IZendeskJiraIntegrations } from "interfaces/integration";
import {
  IPolicyStats,
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
  IPoliciesCountResponse,
  IPolicy,
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
import TableCount from "components/TableContainer/TableCount";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";

import PoliciesTable from "./components/PoliciesTable";
import OtherWorkflowsModal from "./components/OtherWorkflowsModal";
import AddPolicyModal from "./components/AddPolicyModal";
import DeletePolicyModal from "./components/DeletePolicyModal";
import CalendarEventsModal from "./components/CalendarEventsModal";
import { ICalendarEventsFormData } from "./components/CalendarEventsModal/CalendarEventsModal";

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

  const [isUpdatingPolicies, setIsUpdatingPolicies] = useState(false);
  const [isUpdatingCalendarEvents, setIsUpdatingCalendarEvents] = useState(
    false
  );
  const [isUpdatingOtherWorkflows, setIsUpdatingOtherWorkflows] = useState(
    false
  );
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showDeletePolicyModal, setShowDeletePolicyModal] = useState(false);
  const [showCalendarEventsModal, setShowCalendarEventsModal] = useState(false);
  const [showOtherWorkflowsModal, setShowOtherWorkflowsModal] = useState(false);
  const [
    policiesAvailableToAutomate,
    setPoliciesAvailableToAutomate,
  ] = useState<IPolicyStats[]>([]);
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);

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

  // Needs update on location change or table state might not match URL
  const [searchQuery, setSearchQuery] = useState(initialSearchQuery);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(initialSortDirection);

  useEffect(() => {
    setLastEditedQueryPlatform(null);
  }, []);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    setSearchQuery(initialSearchQuery);
    setSortHeader(initialSortHeader);
    setSortDirection(initialSortDirection);
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
      select: (data) => data.policies || [],
      staleTime: 5000,
      onSuccess: (data) => {
        setPoliciesAvailableToAutomate(data || []);
      },
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
        query: isAnyTeamSelected ? "" : searchQuery,
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
        page: tableQueryData?.pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        teamId: teamIdForApi || 0,
        mergeInherited: !!teamIdForApi,
      },
    ],
    ({ queryKey }) => {
      return teamPoliciesAPI.loadAllNew(queryKey[0]);
    },
    {
      enabled: isRouteOk && isPremiumTier && !!teamIdForApi,
      select: (data: ILoadTeamPoliciesResponse) => data.policies || [],
      onSuccess: (data) => {
        const allPoliciesAvailableToAutomate = data.filter(
          (policy: IPolicy) => policy.team_id === currentTeamId
        );
        setPoliciesAvailableToAutomate(allPoliciesAvailableToAutomate || []);
      },
    }
  );

  const {
    data: teamPoliciesCountMergeInherited,
    isFetching: isFetchingTeamCountMergeInherited,
    refetch: refetchTeamPoliciesCountMergeInherited,
  } = useQuery<
    IPoliciesCountResponse,
    Error,
    number,
    ITeamPoliciesCountQueryKey[]
  >(
    [
      {
        scope: "teamPoliciesCountMergeInherited",
        query: searchQuery,
        teamId: teamIdForApi || 0, // TODO: Fix number/undefined type
        mergeInherited: !!teamIdForApi,
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
    if (teamId) {
      refetchTeamPolicies();
      refetchTeamPoliciesCountMergeInherited();
    } else {
      refetchGlobalPolicies(); // Only call on global policies as this is expensive
      refetchGlobalPoliciesCount();
    }
  };

  // NOTE: used to reset page number to 0 when modifying filters
  // NOTE: Solution reused from ManageHostPage.tsx
  useEffect(() => {
    setResetPageIndex(false);
  }, [page]);

  // NOTE: used to reset page number to 0 when modifying filters
  const handleResetPageIndex = () => {
    setTableQueryData(
      (prevState) =>
        ({
          ...prevState,
          pageIndex: 0,
        } as ITableQueryData)
    );
    setResetPageIndex(true);
  };

  const onTeamChange = useCallback(
    (teamId: number) => {
      setSelectedPolicyIds([]);
      handleTeamChange(teamId);
      handleResetPageIndex();
    },
    [handleTeamChange]
  );

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      if (!isRouteOk || isEqual(newTableQuery, tableQueryData)) {
        return;
      }

      setTableQueryData({ ...newTableQuery });

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
        newQueryParams.team_id = teamIdForApi;
      }

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_POLICIES,
        queryParams: { ...queryParams, ...newQueryParams },
      });

      router?.replace(locationPath);
    },
    [isRouteOk, teamIdForApi, searchQuery, sortDirection, page, sortHeader] // Other dependencies can cause infinite re-renders as URL is source of truth
  );

  const toggleOtherWorkflowsModal = () =>
    setShowOtherWorkflowsModal(!showOtherWorkflowsModal);

  const toggleAddPolicyModal = () => setShowAddPolicyModal(!showAddPolicyModal);

  const toggleDeletePolicyModal = () =>
    setShowDeletePolicyModal(!showDeletePolicyModal);

  const toggleCalendarEventsModal = () => {
    setShowCalendarEventsModal(!showCalendarEventsModal);
  };

  const onSelectAutomationOption = (option: string) => {
    switch (option) {
      case "calendar_events":
        toggleCalendarEventsModal();
        break;
      case "other_workflows":
        toggleOtherWorkflowsModal();
        break;
      default:
    }
  };

  const onUpdateOtherWorkflows = async (requestBody: {
    webhook_settings: Pick<IWebhookSettings, "failing_policies_webhook">;
    integrations: IZendeskJiraIntegrations;
  }) => {
    setIsUpdatingOtherWorkflows(true);
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
      toggleOtherWorkflowsModal();
      setIsUpdatingOtherWorkflows(false);
      isAnyTeamSelected ? refetchTeamConfig() : refetchConfig();
    }
  };

  // TODO - finalize

  // const onUpdatePolicySoftwareInstall = async (
  //   formData: IPolicySWInstallFormData
  //   // {policyIDs: swTitleIds}
  // ) => {
  //   setIsUpdatingPolicySoftwareInstall(true);
  //   // get policyIds: swTitleIds that have changed
  //   const changedPolicies = []; // TODO

  //   // if there are any:
  //   try {
  //     const responses: Promise<any>[] = [];
  //     responses.concat(
  //       changedPolicies.map((changedPolicy) => {
  //         return teamPoliciesAPI.update(changedPolicy.id, {
  //           software_title_id: changedPolicy.software_title_id || null, // TODO: confirm undefined/null
  //           team_id: teamIdForApi,
  //         });
  //       })
  //     );

  //     await Promise.all(responses);
  //     await wait(100); // Wait 100ms to avoid race conditions with refetch
  //     await refetchTeamPolicies();
  //     renderFlash("success", "Successfully updated policy automations.");
  //   } catch {
  //     renderFlash(
  //       "error",
  //       "Could not update policy automations. Please try again."
  //     );
  //   } finally {
  //     togglePolicySoftwareInstallModal();
  //     setIsUpdatingPolicySoftwareInstall(false);
  //   }
  // };

  const onUpdateCalendarEvents = async (formData: ICalendarEventsFormData) => {
    setIsUpdatingCalendarEvents(true);

    try {
      // update team config if either field has been changed
      const responses: Promise<any>[] = [];
      if (
        formData.enabled !==
          teamConfig?.integrations.google_calendar?.enable_calendar_events ||
        formData.url !== teamConfig?.integrations.google_calendar?.webhook_url
      ) {
        responses.push(
          teamsAPI.update(
            {
              integrations: {
                google_calendar: {
                  enable_calendar_events: formData.enabled,
                  webhook_url: formData.url,
                },
                // These fields will never actually be changed here. See comment above
                // IGlobalIntegrations definition.
                zendesk: teamConfig?.integrations.zendesk || [],
                jira: teamConfig?.integrations.jira || [],
              },
            },
            teamIdForApi
          )
        );
      }

      // update changed policies calendar events enabled
      const changedPolicies = formData.policies.filter((formPolicy) => {
        const prevPolicyState = policiesAvailableToAutomate.find(
          (policy) => policy.id === formPolicy.id
        );
        return (
          formPolicy.isChecked !== prevPolicyState?.calendar_events_enabled
        );
      });

      responses.concat(
        changedPolicies.map((changedPolicy) => {
          return teamPoliciesAPI.update(changedPolicy.id, {
            calendar_events_enabled: changedPolicy.isChecked,
            team_id: teamIdForApi,
          });
        })
      );

      await Promise.all(responses);
      await wait(100); // Wait 100ms to avoid race conditions with refetch
      await refetchTeamPolicies();
      await refetchTeamConfig();

      renderFlash("success", "Successfully updated policy automations.");
    } catch {
      renderFlash(
        "error",
        "Could not update policy automations. Please try again."
      );
    } finally {
      toggleCalendarEventsModal();
      setIsUpdatingCalendarEvents(false);
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

  const policiesErrors = isAnyTeamSelected
    ? teamPoliciesError
    : globalPoliciesError;

  const policyResults = isAnyTeamSelected
    ? teamPolicies && teamPolicies.length > 0
    : globalPolicies && globalPolicies.length > 0;

  // Show CTA buttons if there is no errors AND there are policy results or a search filter
  const showCtaButtons =
    !policiesErrors && (policyResults || searchQuery !== "");

  const automationsConfig = isAnyTeamSelected ? teamConfig : config;
  const hasPoliciesToAutomateOrDelete = policiesAvailableToAutomate.length > 0;
  const showAutomationsDropdown =
    canManageAutomations && automationsConfig && hasPoliciesToAutomateOrDelete;

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
    // Hide count if fetching count || there are errors OR there are no policy results with no a search filter
    const isFetchingCount = isAnyTeamSelected
      ? isFetchingTeamCountMergeInherited
      : isFetchingGlobalCount;

    const hideCount =
      isFetchingCount ||
      policiesErrors ||
      (!policyResults && searchQuery === "");

    if (hideCount) {
      return null;
    }

    return <TableCount name="policies" count={count} />;
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
            hasPoliciesToDelete={hasPoliciesToAutomateOrDelete}
            currentTeam={currentTeamSummary}
            currentAutomatedPolicies={currentAutomatedPolicies}
            renderPoliciesCount={() =>
              renderPoliciesCount(teamPoliciesCountMergeInherited)
            }
            isPremiumTier={isPremiumTier}
            searchQuery={searchQuery}
            sortHeader={sortHeader}
            sortDirection={sortDirection}
            page={page}
            onQueryChange={onQueryChange}
            resetPageIndex={resetPageIndex}
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
            hasPoliciesToDelete={hasPoliciesToAutomateOrDelete}
            currentTeam={currentTeamSummary}
            currentAutomatedPolicies={currentAutomatedPolicies}
            isPremiumTier={isPremiumTier}
            renderPoliciesCount={() => renderPoliciesCount(globalPoliciesCount)}
            searchQuery={searchQuery}
            sortHeader={sortHeader}
            sortDirection={sortDirection}
            page={page}
            onQueryChange={onQueryChange}
            resetPageIndex={resetPageIndex}
          />
        )}
      </div>
    );
  };

  const getAutomationsDropdownOptions = () => {
    const isAllTeams = teamIdForApi === undefined || teamIdForApi === -1;
    let disabledTooltipContent: React.ReactNode;
    if (!isPremiumTier) {
      disabledTooltipContent = "Available in Fleet Premium.";
    } else if (isAllTeams) {
      disabledTooltipContent = (
        <>
          Select a team to manage
          <br />
          calendar events.
        </>
      );
    }

    return [
      {
        label: "Calendar events",
        value: "calendar_events",
        disabled: !isPremiumTier || isAllTeams,
        helpText: "Automatically reserve time to resolve failing policies.",
        tooltipContent: disabledTooltipContent,
      },
      {
        label: "Other workflows",
        value: "other_workflows",
        disabled: false,
        helpText: "Create tickets or fire webhooks for failing policies.",
      },
    ];
  };

  const isCalEventsConfigured =
    (config?.integrations.google_calendar &&
      config?.integrations.google_calendar.length > 0) ??
    false;

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
              {showAutomationsDropdown && (
                <div className={`${baseClass}__manage-automations-wrapper`}>
                  <Dropdown
                    className={`${baseClass}__manage-automations-dropdown`}
                    onChange={onSelectAutomationOption}
                    placeholder="Manage automations"
                    searchable={false}
                    options={getAutomationsDropdownOptions()}
                  />
                </div>
              )}
              {canAddOrDeletePolicy && (
                <div className={`${baseClass}__action-button-container`}>
                  <Button
                    variant="brand"
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
        <div className={`${baseClass}__description`}>
          <p>
            {isAnyTeamSelected
              ? "Detect device health issues for all hosts assigned to this team."
              : "Detect device health issues for all hosts."}
          </p>
        </div>
        {renderMainTable()}
        {config && automationsConfig && showOtherWorkflowsModal && (
          <OtherWorkflowsModal
            automationsConfig={automationsConfig}
            availableIntegrations={config.integrations}
            availablePolicies={policiesAvailableToAutomate}
            isUpdating={isUpdatingOtherWorkflows}
            onExit={toggleOtherWorkflowsModal}
            onSubmit={onUpdateOtherWorkflows}
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
        {showCalendarEventsModal && (
          <CalendarEventsModal
            onExit={toggleCalendarEventsModal}
            onSubmit={onUpdateCalendarEvents}
            configured={isCalEventsConfigured}
            enabled={
              teamConfig?.integrations.google_calendar
                ?.enable_calendar_events ?? false
            }
            url={teamConfig?.integrations.google_calendar?.webhook_url || ""}
            policies={policiesAvailableToAutomate}
            isUpdating={isUpdatingCalendarEvents}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManagePolicyPage;
