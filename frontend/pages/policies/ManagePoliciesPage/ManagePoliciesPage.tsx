// TODO: make 'queryParams', 'router', and 'tableQueryData' dependencies stable (aka, memoized)
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
import { API_ALL_TEAMS_ID, API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";

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
import InstallSoftwareModal from "./components/InstallSoftwareModal";
import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";
import PolicyRunScriptModal from "./components/PolicyRunScriptModal";
import { IPolicyRunScriptFormData } from "./components/PolicyRunScriptModal/PolicyRunScriptModal";

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
const [
  DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG,
  DEFAULT_AUTOMATION_UPDATE_ERR_MSG,
] = [
  "Successfully updated policy automations.",
  "Could not update policy automations. Please try again.",
];

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
    },
  });

  // loading state used by various policy updates on this page
  const [isUpdatingPolicies, setIsUpdatingPolicies] = useState(false);

  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showDeletePolicyModal, setShowDeletePolicyModal] = useState(false);
  const [showInstallSoftwareModal, setShowInstallSoftwareModal] = useState(
    false
  );
  const [showPolicyRunScriptModal, setShowPolicyRunScriptModal] = useState(
    false
  );
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
  }, [setLastEditedQueryPlatform]);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    setSearchQuery(initialSearchQuery);
    setSortHeader(initialSortHeader);
    setSortDirection(initialSortDirection);
  }, [
    location,
    isRouteOk,
    initialSearchQuery,
    initialSortHeader,
    initialSortDirection,
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
      enabled: isRouteOk && isAllTeamsSelected,
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
        query: !isAllTeamsSelected ? "" : searchQuery,
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
        page: tableQueryData?.pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        orderKey: sortHeader,
        // teamIdForApi will never actually be undefined here
        teamId: teamIdForApi || 0,
        // no teams does inherit
        mergeInherited: true,
      },
    ],
    ({ queryKey }) => {
      return teamPoliciesAPI.loadAllNew(queryKey[0]);
    },
    {
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
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
      enabled: isRouteOk && isPremiumTier && !isAllTeamsSelected,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  const canAddOrDeletePolicy =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

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
      enabled: isRouteOk && canAddOrDeletePolicy,
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
      // no call for no team (teamIdForApi === 0)
      enabled: isRouteOk && !!teamIdForApi && canAddOrDeletePolicy,
      select: (data) => data.team,
    }
  );

  const refetchPolicies = (teamId?: number) => {
    if (teamId !== undefined) {
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
  }, []);

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
    [
      isRouteOk,
      tableQueryData,
      sortDirection,
      sortHeader,
      searchQuery,
      teamIdForApi,
      queryParams,
      router,
    ] // Other dependencies can cause infinite re-renders as URL is source of truth
  );

  const toggleOtherWorkflowsModal = () =>
    setShowOtherWorkflowsModal(!showOtherWorkflowsModal);

  const toggleAddPolicyModal = () => setShowAddPolicyModal(!showAddPolicyModal);

  const toggleDeletePolicyModal = () =>
    setShowDeletePolicyModal(!showDeletePolicyModal);

  const toggleInstallSoftwareModal = () => {
    setShowInstallSoftwareModal(!showInstallSoftwareModal);
  };

  const togglePolicyRunScriptModal = () => {
    setShowPolicyRunScriptModal(!showPolicyRunScriptModal);
  };

  const toggleCalendarEventsModal = () => {
    setShowCalendarEventsModal(!showCalendarEventsModal);
  };

  const onSelectAutomationOption = (option: string) => {
    switch (option) {
      case "calendar_events":
        toggleCalendarEventsModal();
        break;
      case "install_software":
        toggleInstallSoftwareModal();
        break;
      case "run_script":
        togglePolicyRunScriptModal();
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
    setIsUpdatingPolicies(true);
    try {
      await (!isAllTeamsSelected
        ? teamsAPI.update(requestBody, teamIdForApi)
        : configAPI.update(requestBody));
      renderFlash("success", DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG);
    } catch {
      renderFlash("error", DEFAULT_AUTOMATION_UPDATE_ERR_MSG);
    } finally {
      toggleOtherWorkflowsModal();
      setIsUpdatingPolicies(false);
      !isAllTeamsSelected ? refetchTeamConfig() : refetchConfig();
    }
  };

  const onUpdatePolicySoftwareInstall = async (
    formData: IInstallSoftwareFormData
  ) => {
    try {
      setIsUpdatingPolicies(true);
      const changedPolicies = formData.filter((formPolicy) => {
        const prevPolicyState = policiesAvailableToAutomate.find(
          (policy) => policy.id === formPolicy.id
        );

        const turnedOff =
          prevPolicyState?.install_software !== undefined &&
          formPolicy.installSoftwareEnabled === false;

        const turnedOn =
          prevPolicyState?.install_software === undefined &&
          formPolicy.installSoftwareEnabled === true;

        const updatedSwId =
          prevPolicyState?.install_software?.software_title_id !== undefined &&
          formPolicy.swIdToInstall !==
            prevPolicyState?.install_software?.software_title_id;

        return turnedOff || turnedOn || updatedSwId;
      });
      if (!changedPolicies.length) {
        renderFlash("success", "No changes detected.");
        return;
      }
      const responses: Promise<
        ReturnType<typeof teamPoliciesAPI.update>
      >[] = [];
      responses.concat(
        changedPolicies.map((changedPolicy) => {
          return teamPoliciesAPI.update(changedPolicy.id, {
            // "software_title_id": 0 will unset software install for the policy
            // "software_title_id": X will set the value to the given integer (except 0).
            software_title_id: changedPolicy.swIdToInstall || 0,
            team_id: teamIdForApi,
          });
        })
      );
      await Promise.all(responses);
      await wait(100); // prevent race
      refetchTeamPolicies();
      renderFlash("success", DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG);
    } catch {
      renderFlash("error", DEFAULT_AUTOMATION_UPDATE_ERR_MSG);
    } finally {
      toggleInstallSoftwareModal();
      setIsUpdatingPolicies(false);
    }
  };

  const onUpdatePolicyRunScript = async (
    formData: IPolicyRunScriptFormData
  ) => {
    try {
      setIsUpdatingPolicies(true);
      const changedPolicies = formData.filter((formPolicy) => {
        const prevPolicyState = policiesAvailableToAutomate.find(
          (policy) => policy.id === formPolicy.id
        );

        const turnedOff =
          prevPolicyState?.run_script !== undefined &&
          formPolicy.runScriptEnabled === false;

        const turnedOn =
          prevPolicyState?.run_script === undefined &&
          formPolicy.runScriptEnabled === true;

        const updatedRunScriptId =
          prevPolicyState?.run_script?.id !== undefined &&
          formPolicy.scriptIdToRun !== prevPolicyState?.run_script?.id;

        return turnedOff || turnedOn || updatedRunScriptId;
      });
      if (!changedPolicies.length) {
        renderFlash("success", "No changes detected.");
        return;
      }
      const responses: Promise<
        ReturnType<typeof teamPoliciesAPI.update>
      >[] = [];
      responses.concat(
        changedPolicies.map((changedPolicy) => {
          return teamPoliciesAPI.update(changedPolicy.id, {
            // "script_id": 0 will unset running a script for the policy (a script never has ID 0)
            // "script_id": X will sets script X to run when the policy fails
            script_id: changedPolicy.scriptIdToRun || 0,
            team_id: teamIdForApi,
          });
        })
      );
      await Promise.all(responses);
      await wait(100);
      refetchTeamPolicies();
      renderFlash("success", DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG);
    } catch {
      renderFlash("error", DEFAULT_AUTOMATION_UPDATE_ERR_MSG);
    } finally {
      togglePolicyRunScriptModal();
      setIsUpdatingPolicies(false);
    }
  };

  const onUpdateCalendarEvents = async (formData: ICalendarEventsFormData) => {
    setIsUpdatingPolicies(true);

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

      renderFlash("success", DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG);
    } catch {
      renderFlash("error", DEFAULT_AUTOMATION_UPDATE_ERR_MSG);
    } finally {
      toggleCalendarEventsModal();
      setIsUpdatingPolicies(false);
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
      const request = !isAllTeamsSelected
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

  const policiesErrors = !isAllTeamsSelected
    ? teamPoliciesError
    : globalPoliciesError;

  const policyResults = !isAllTeamsSelected
    ? teamPolicies && teamPolicies.length > 0
    : globalPolicies && globalPolicies.length > 0;

  // Show CTA buttons if there is no errors AND there are policy results or a search filter
  const showCtaButtons =
    !policiesErrors && (policyResults || searchQuery !== "");

  const automationsConfig = !isAllTeamsSelected ? teamConfig : config;
  const hasPoliciesToAutomateOrDelete = policiesAvailableToAutomate.length > 0;
  const showAutomationsDropdown =
    canManageAutomations && hasPoliciesToAutomateOrDelete;

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
    const isFetchingCount = !isAllTeamsSelected
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
    if (!isRouteOk || (isPremiumTier && !userTeams)) {
      return <Spinner />;
    }
    if (isAllTeamsSelected) {
      // Global policies

      if (globalPoliciesError) {
        return <TableDataError />;
      }
      return (
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
      );
    }

    // Team policies
    if (teamPoliciesError) {
      return <TableDataError />;
    }
    return (
      <div>
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
      </div>
    );
  };

  const getAutomationsDropdownOptions = (configPresent: boolean) => {
    let disabledInstallTooltipContent: React.ReactNode;
    let disabledCalendarTooltipContent: React.ReactNode;
    let disabledRunScriptTooltipContent: React.ReactNode;
    if (!isPremiumTier) {
      disabledInstallTooltipContent = "Available in Fleet Premium.";
      disabledCalendarTooltipContent = "Available in Fleet Premium.";
      disabledRunScriptTooltipContent = "Available in Fleet Premium.";
    } else if (isAllTeamsSelected) {
      disabledInstallTooltipContent = (
        <>
          Select a team to manage
          <br />
          install software automation.
        </>
      );
      disabledCalendarTooltipContent = (
        <>
          Select a team to manage
          <br />
          calendar events.
        </>
      );
      disabledRunScriptTooltipContent = (
        <>
          Select a team to manage
          <br />
          run script automation.
        </>
      );
    }
    const installSWOption = {
      label: "Install software",
      value: "install_software",
      disabled: !!disabledInstallTooltipContent,
      helpText: "Install software to resolve failing policies.",
      tooltipContent: disabledInstallTooltipContent,
    };
    const runScriptOption = {
      label: "Run script",
      value: "run_script",
      disabled: !!disabledRunScriptTooltipContent,
      helpText: "Run script to resolve failing policies.",
      tooltipContent: disabledRunScriptTooltipContent,
    };

    // Maintainers do not have access to automate calendar events or other workflows
    // Config must be present to update calendar events or other workflows
    if (!configPresent || isGlobalMaintainer || isTeamMaintainer) {
      return [installSWOption, runScriptOption];
    }

    return [
      {
        label: "Calendar events",
        value: "calendar_events",
        disabled: !!disabledCalendarTooltipContent,
        helpText: "Automatically reserve time to resolve failing policies.",
        tooltipContent: disabledCalendarTooltipContent,
      },
      installSWOption,
      runScriptOption,
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

  if (!isRouteOk) {
    return <Spinner />;
  }

  let teamsDropdownHelpText: string;
  if (teamIdForApi === API_NO_TEAM_ID) {
    teamsDropdownHelpText =
      "Detect device health issues for hosts that are not on a team.";
  } else if (teamIdForApi === API_ALL_TEAMS_ID) {
    teamsDropdownHelpText = "Detect device health issues for all hosts.";
  } else {
    // a team is selected
    teamsDropdownHelpText =
      "Detect device health issues for all hosts assigned to this team.";
  }
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
                      includeNoTeams
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
                    options={getAutomationsDropdownOptions(!!automationsConfig)}
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
          <p>{teamsDropdownHelpText}</p>
        </div>
        {renderMainTable()}
        {config && automationsConfig && showOtherWorkflowsModal && (
          <OtherWorkflowsModal
            automationsConfig={automationsConfig}
            availableIntegrations={config.integrations}
            availablePolicies={policiesAvailableToAutomate}
            isUpdating={isUpdatingPolicies}
            onExit={toggleOtherWorkflowsModal}
            onSubmit={onUpdateOtherWorkflows}
          />
        )}
        {showAddPolicyModal && (
          <AddPolicyModal
            onCancel={toggleAddPolicyModal}
            router={router}
            // default to all teams, though should be present here
            teamId={currentTeamId ?? API_ALL_TEAMS_ID}
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
        {showInstallSoftwareModal && (
          <InstallSoftwareModal
            onExit={toggleInstallSoftwareModal}
            onSubmit={onUpdatePolicySoftwareInstall}
            isUpdating={isUpdatingPolicies}
            policies={policiesAvailableToAutomate}
            // currentTeamId will at this point be present
            teamId={currentTeamId ?? 0}
          />
        )}
        {showPolicyRunScriptModal && (
          <PolicyRunScriptModal
            onExit={togglePolicyRunScriptModal}
            onSubmit={onUpdatePolicyRunScript}
            isUpdating={isUpdatingPolicies}
            policies={policiesAvailableToAutomate}
            // currentTeamId will at this point be present
            teamId={currentTeamId ?? 0}
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
            isUpdating={isUpdatingPolicies}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManagePolicyPage;
