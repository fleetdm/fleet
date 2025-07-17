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
import { INotification } from "interfaces/notification";
import {
  IPolicyStats,
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
  IPoliciesCountResponse,
  IPolicy,
} from "interfaces/policy";
import {
  API_ALL_TEAMS_ID,
  API_NO_TEAM_ID,
  APP_CONTEXT_ALL_TEAMS_ID,
  ITeamConfig,
} from "interfaces/team";
import { TooltipContent } from "interfaces/dropdownOption";

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

import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import LastUpdatedText from "components/LastUpdatedText";
import TooltipWrapper from "components/TooltipWrapper";

import PoliciesTable from "./components/PoliciesTable";
import OtherWorkflowsModal from "./components/OtherWorkflowsModal";
import DeletePoliciesModal from "./components/DeletePoliciesModal";
import CalendarEventsModal from "./components/CalendarEventsModal";
import { ICalendarEventsFormData } from "./components/CalendarEventsModal/CalendarEventsModal";
import InstallSoftwareModal from "./components/InstallSoftwareModal";
import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";
import PolicyRunScriptModal from "./components/PolicyRunScriptModal";
import { IPolicyRunScriptFormData } from "./components/PolicyRunScriptModal/PolicyRunScriptModal";
import {
  getInstallSoftwareErrorMessage,
  getRunScriptErrorMessage,
} from "./helpers";
import { DEFAULT_POLICY } from "../constants";
import ConditionalAccessModal from "./components/ConditionalAccessModal";
import { IConditionalAccessFormData } from "./components/ConditionalAccessModal/ConditionalAccessModal";

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

export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 20;
export const DEFAULT_SORT_COLUMN = "name";
const [
  DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG,
  DEFAULT_AUTOMATION_UPDATE_ERR_MSG,
] = [
  "Successfully updated policy automations.",
  "Could not update policy automations.",
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
    config: globalConfigFromContext,
    setConfig,
    setFilteredPoliciesPath,
    filteredPoliciesPath,
  } = useContext(AppContext);
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);
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
  const [showDeletePoliciesModal, setShowDeletePoliciesModal] = useState(false);
  const [showInstallSoftwareModal, setShowInstallSoftwareModal] = useState(
    false
  );
  const [showPolicyRunScriptModal, setShowPolicyRunScriptModal] = useState(
    false
  );
  const [showCalendarEventsModal, setShowCalendarEventsModal] = useState(false);
  const [showOtherWorkflowsModal, setShowOtherWorkflowsModal] = useState(false);
  const [showConditionalAccessModal, setShowConditionalAccessModal] = useState(
    false
  );
  const [
    policiesAvailableToAutomate,
    setPoliciesAvailableToAutomate,
  ] = useState<IPolicyStats[]>([]);

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
  const [
    tableQueryDataForApi,
    setTableQueryDataForApi,
  ] = useState<ITableQueryData>();
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
        page,
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
        page,
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
        mergeInherited: true,
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

  const canAddOrDeletePolicies =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  const {
    data: globalConfig,
    isFetching: isFetchingGlobalConfig,
    refetch: refetchGlobalConfig,
  } = useQuery<IConfig, Error>(
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

  const {
    data: teamConfig,
    isFetching: isFetchingTeamConfig,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teams", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      // no call for no team (teamIdForApi === 0)
      enabled: isRouteOk && !!teamIdForApi && canAddOrDeletePolicies,
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
        newQueryParams.team_id = teamIdForApi;
      }

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_POLICIES,
        queryParams: { ...queryParams, ...newQueryParams },
      });

      router?.push(locationPath);
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

  const toggleOtherWorkflowsModal = () =>
    setShowOtherWorkflowsModal(!showOtherWorkflowsModal);

  const toggleDeletePoliciesModal = () =>
    setShowDeletePoliciesModal(!showDeletePoliciesModal);

  const toggleInstallSoftwareModal = () => {
    setShowInstallSoftwareModal(!showInstallSoftwareModal);
  };

  const togglePolicyRunScriptModal = () => {
    setShowPolicyRunScriptModal(!showPolicyRunScriptModal);
  };

  const toggleCalendarEventsModal = () => {
    setShowCalendarEventsModal(!showCalendarEventsModal);
  };

  const toggleConditionalAccessModal = () => {
    setShowConditionalAccessModal(!showConditionalAccessModal);
  };

  const onSelectAutomationOption = (option: SingleValue<CustomOptionType>) => {
    switch (option?.value) {
      case "calendar_events":
        toggleCalendarEventsModal();
        break;
      case "install_software":
        toggleInstallSoftwareModal();
        break;
      case "run_script":
        togglePolicyRunScriptModal();
        break;
      case "conditional_access":
        toggleConditionalAccessModal();
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
      !isAllTeamsSelected ? refetchTeamConfig() : refetchGlobalConfig();
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

      const promises = changedPolicies.map((changedPolicy) =>
        teamPoliciesAPI.update(changedPolicy.id, {
          software_title_id: changedPolicy.swIdToInstall || null,
          team_id: teamIdForApi,
        })
      );

      // Allows for all API calls to settle even if there is an error on one
      const results = await Promise.allSettled(promises);

      const successfulUpdates = results.filter(
        (result) => result.status === "fulfilled"
      );
      const failedUpdates = results.filter(
        (result) => result.status === "rejected"
      );

      // Renders API error reason for each error in a single message
      if (failedUpdates.length > 0) {
        const errorNotifications: INotification[] = failedUpdates.map(
          (result, index) => {
            const message = getInstallSoftwareErrorMessage(
              result as PromiseRejectedResult,
              formData,
              currentTeamName
            );

            return {
              id: `error-${index}`,
              alertType: "error",
              isVisible: true,
              message,
              persistOnPageChange: false,
            };
          }
        );

        renderMultiFlash({
          notifications: errorNotifications,
        });
      } else if (successfulUpdates.length > 0) {
        // Only render success message if there are no failures
        renderFlash("success", DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG);
      }

      await wait(100); // Wait 100ms to avoid race conditions with refetch
      refetchTeamPolicies();
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
          formPolicy.scriptIdToRun === null;

        const turnedOn =
          prevPolicyState?.run_script === undefined &&
          formPolicy.scriptIdToRun !== null;

        const updatedScriptId =
          prevPolicyState?.run_script?.id !== undefined &&
          formPolicy.scriptIdToRun !== prevPolicyState?.run_script?.id;

        return turnedOff || turnedOn || updatedScriptId;
      });

      if (!changedPolicies.length) {
        renderFlash("success", "No changes detected.");
        return;
      }

      const promises = formData.map((changedPolicy) =>
        teamPoliciesAPI.update(changedPolicy.id, {
          // "script_id": null will unset running a script for the policy
          // "script_id": X will sets script X to run when the policy fails
          script_id: changedPolicy.scriptIdToRun || null,
          team_id: teamIdForApi,
        })
      );

      // Allows for all API calls to settle even if there is an error on one
      const results = await Promise.allSettled(promises);

      const successfulUpdates = results.filter(
        (result) => result.status === "fulfilled"
      );
      const failedUpdates = results.filter(
        (result) => result.status === "rejected"
      );

      // Renders API error reason for each error in a single message
      if (failedUpdates.length > 0) {
        const errorNotifications: INotification[] = failedUpdates.map(
          (result, index) => {
            const message = getRunScriptErrorMessage(
              result as PromiseRejectedResult,
              formData,
              currentTeamName
            );

            return {
              id: `error-${index}`,
              alertType: "error",
              isVisible: true,
              message,
              persistOnPageChange: false,
            };
          }
        );

        renderMultiFlash({
          notifications: errorNotifications,
        });
      } else if (successfulUpdates.length > 0) {
        // Only render success message if there are no failures
        renderFlash("success", DEFAULT_AUTOMATION_UPDATE_SUCCESS_MSG);
      }

      await wait(100); // Wait 100ms to avoid race conditions with refetch
      refetchTeamPolicies();
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
      responses.concat(
        formData.changedPolicies.map((changedPolicy) => {
          return teamPoliciesAPI.update(changedPolicy.id, {
            calendar_events_enabled: changedPolicy.calendar_events_enabled,
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

  const onUpdateConditionalAccess = async ({
    enabled: enableConditionalAccess,
    changedPolicies,
  }: IConditionalAccessFormData) => {
    setIsUpdatingPolicies(true);

    try {
      // TODO - narrow type for update policy responses
      const responses: (Promise<ITeamConfig> | Promise<any>)[] = [];
      let refetchConfig: any;

      // If enabling/disabling the feature, update appropriate config
      if (teamIdForApi === API_NO_TEAM_ID) {
        if (
          enableConditionalAccess !==
          globalConfig?.integrations.conditional_access_enabled
        ) {
          const payload = {
            integrations: {
              conditional_access_enabled: enableConditionalAccess,
            },
          };
          responses.push(configAPI.update(payload));
          refetchConfig = refetchGlobalConfig;
        }
      } else if (
        enableConditionalAccess !==
        teamConfig?.integrations.conditional_access_enabled
      ) {
        // patch team config (all teams but No team)
        const payload = {
          integrations: {
            // These fields will never actually be changed here. See comment above
            // IGlobalIntegrations definition.
            zendesk: teamConfig?.integrations.zendesk || [],
            jira: teamConfig?.integrations.jira || [],
            conditional_access_enabled: enableConditionalAccess,
          },
        };
        responses.push(teamsAPI.update(payload, teamIdForApi));
        refetchConfig = refetchTeamConfig;
      }

      // handle any changed policies for no team or a team
      responses.concat(
        changedPolicies.map((changedPolicy) => {
          return teamPoliciesAPI.update(changedPolicy.id, {
            conditional_access_enabled:
              changedPolicy.conditional_access_enabled,
            team_id: teamIdForApi,
          });
        })
      );
      await Promise.all(responses);
      await wait(100); // helps avoid refetch race conditions
      if (refetchConfig) {
        await refetchConfig();
      }
      renderFlash(
        "success",
        "Successfully updated conditional access automations."
      );
    } catch {
      renderFlash("error", "Could not update conditional access automations.");
    } finally {
      toggleConditionalAccessModal();
      setIsUpdatingPolicies(false);
    }
  };

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
        : `${PATHS.NEW_POLICY}?team_id=${currentTeamId}`
    );
  };

  const onDeletePoliciesClick = (selectedTableIds: number[]): void => {
    toggleDeletePoliciesModal();
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
      toggleDeletePoliciesModal();
      setIsUpdatingPolicies(false);
    }
  };

  const policiesErrors = !isAllTeamsSelected
    ? teamPoliciesError
    : globalPoliciesError;

  const policyResults = !isAllTeamsSelected
    ? teamPolicies && teamPolicies.length > 0
    : globalPolicies && globalPolicies.length > 0;

  // Show CTA buttons if there are no errors
  const showCtaButtons = !policiesErrors;

  const automationsConfig = !isAllTeamsSelected ? teamConfig : globalConfig;
  const hasPoliciesToAutomateOrDelete = policiesAvailableToAutomate.length > 0;
  const showAutomationsDropdown = canManageAutomations;

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

  const renderPoliciesCountAndLastUpdated = (
    count?: number,
    policies?: IPolicyStats[]
  ) => {
    // Hide count if fetching count || there are errors OR there are no policy results with no a search filter
    const isFetchingCount = !isAllTeamsSelected
      ? isFetchingTeamCountMergeInherited
      : isFetchingGlobalCount;

    const hide =
      isFetchingCount ||
      policiesErrors ||
      (!policyResults && searchQuery === "");

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
          canAddOrDeletePolicies={canAddOrDeletePolicies}
          hasPoliciesToDelete={hasPoliciesToAutomateOrDelete}
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
        />
      );
    }

    // Team policies
    if (teamPoliciesError) {
      return <TableDataError verticalPaddingSize="pad-xxxlarge" />;
    }
    return (
      <div>
        <PoliciesTable
          policiesList={teamPolicies || []}
          isLoading={
            isFetchingTeamPolicies ||
            isFetchingTeamConfig ||
            isFetchingGlobalConfig
          }
          onDeletePoliciesClick={onDeletePoliciesClick}
          canAddOrDeletePolicies={canAddOrDeletePolicies}
          hasPoliciesToDelete={hasPoliciesToAutomateOrDelete}
          currentTeam={currentTeamSummary}
          currentAutomatedPolicies={currentAutomatedPolicies}
          renderPoliciesCount={() =>
            renderPoliciesCountAndLastUpdated(
              teamPoliciesCountMergeInherited,
              teamPolicies
            )
          }
          isPremiumTier={isPremiumTier}
          count={teamPoliciesCountMergeInherited || 0}
          searchQuery={searchQuery}
          sortHeader={sortHeader}
          sortDirection={sortDirection}
          page={page}
          onQueryChange={onQueryChange}
        />
      </div>
    );
  };

  const gitOpsModeEnabled = globalConfig?.gitops.gitops_mode_enabled;

  const isCalEventsConfigured =
    (globalConfig?.integrations.google_calendar &&
      globalConfig?.integrations.google_calendar.length > 0) ??
    false;

  const isCalEventsEnabled =
    teamConfig?.integrations.google_calendar?.enable_calendar_events ?? false;

  const isConditionalAccessConfigured =
    globalConfig?.conditional_access?.microsoft_entra_connection_configured ??
    false;

  const isConditionalAccessEnabled =
    (teamIdForApi === API_NO_TEAM_ID
      ? globalConfig?.integrations.conditional_access_enabled
      : teamConfig?.integrations.conditional_access_enabled) ?? false;

  const getAutomationsDropdownOptions = (configPresent: boolean) => {
    let disabledInstallTooltipContent: TooltipContent;
    let disabledCalendarTooltipContent: TooltipContent;
    let disabledRunScriptTooltipContent: TooltipContent;
    let disabledConditionalAccessTooltipContent: TooltipContent;
    if (!isPremiumTier) {
      disabledInstallTooltipContent = "Available in Fleet Premium";
      disabledCalendarTooltipContent = "Available in Fleet Premium";
      disabledRunScriptTooltipContent = "Available in Fleet Premium";
      disabledConditionalAccessTooltipContent = "Available in Fleet Premium";
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
      disabledConditionalAccessTooltipContent = (
        <>
          Select a team to manage
          <br />
          conditional access.
        </>
      );
    } else if (
      (isGlobalMaintainer || isTeamMaintainer) &&
      !isCalEventsEnabled
    ) {
      disabledCalendarTooltipContent = (
        <>
          Contact a user with an
          <br />
          admin role for access.
        </>
      );
    }

    const options: CustomOptionType[] = [
      {
        label: "Calendar",
        value: "calendar_events",
        isDisabled: !!disabledCalendarTooltipContent,
        helpText: "Automatically reserve time to resolve failing policies.",
        tooltipContent: disabledCalendarTooltipContent,
      },
      {
        label: "Software",
        value: "install_software",
        isDisabled: !!disabledInstallTooltipContent,
        helpText: "Install software to resolve failing policies.",
        tooltipContent: disabledInstallTooltipContent,
      },
      {
        label: "Scripts",
        value: "run_script",
        isDisabled: !!disabledRunScriptTooltipContent,
        helpText: "Run script to resolve failing policies.",
        tooltipContent: disabledRunScriptTooltipContent,
      },
    ];

    if (globalConfigFromContext?.license.managed_cloud) {
      options.push({
        label: "Conditional access",
        value: "conditional_access",
        isDisabled: !!disabledConditionalAccessTooltipContent,
        helpText: "Block single sign-on for hosts failing policies.",
        tooltipContent: disabledConditionalAccessTooltipContent,
      });
    }

    // Maintainers do not have access to other workflows
    if (configPresent && !isGlobalMaintainer && !isTeamMaintainer) {
      options.push({
        label: "Other",
        value: "other_workflows",
        isDisabled: false,
        helpText: "Create tickets or fire webhooks for failing policies.",
      });
    }

    return options;
  };

  let automationsDropdown = null;
  if (showAutomationsDropdown) {
    automationsDropdown = (
      <div className={`${baseClass}__manage-automations-wrapper`}>
        <DropdownWrapper
          isDisabled={!hasPoliciesToAutomateOrDelete}
          className={`${baseClass}__manage-automations-dropdown`}
          name="policy-automations"
          onChange={onSelectAutomationOption}
          placeholder="Manage automations"
          options={
            hasPoliciesToAutomateOrDelete
              ? getAutomationsDropdownOptions(!!automationsConfig)
              : []
          }
          variant="button"
          nowrapMenu
        />
      </div>
    );
    if (!hasPoliciesToAutomateOrDelete) {
      const tipContent =
        isPremiumTier &&
        currentTeamId !== APP_CONTEXT_ALL_TEAMS_ID &&
        !globalConfigFromContext?.partnerships?.enable_primo ? (
          <div className={`${baseClass}__header__tooltip`}>
            To manage automations add a policy to this team.
            <br />
            For inherited policies select &ldquo;All teams&rdquo;.
          </div>
        ) : (
          <div className={`${baseClass}__header__tooltip`}>
            To manage automations add a policy.
          </div>
        );

      automationsDropdown = (
        <TooltipWrapper
          underline={false}
          tipContent={tipContent}
          position="top"
          showArrow
        >
          {automationsDropdown}
        </TooltipWrapper>
      );
    }
  }

  if (!isRouteOk) {
    return <Spinner />;
  }

  const renderHeader = () => {
    if (isPremiumTier && !globalConfigFromContext?.partnerships?.enable_primo) {
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

  let teamsDropdownHelpText: string;
  if (teamIdForApi === API_NO_TEAM_ID) {
    teamsDropdownHelpText = `Detect device health issues${
      globalConfigFromContext?.partnerships?.enable_primo
        ? ""
        : " for hosts that are not on a team"
    }.`;
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
              <div className={`${baseClass}__title`}>{renderHeader()}</div>
            </div>
          </div>
          {showCtaButtons && (
            <div className={`${baseClass} button-wrap`}>
              {automationsDropdown}
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
        <div className={`${baseClass}__description`}>
          <p>{teamsDropdownHelpText}</p>
        </div>
        {renderMainTable()}
        {globalConfig && automationsConfig && showOtherWorkflowsModal && (
          <OtherWorkflowsModal
            automationsConfig={automationsConfig}
            availableIntegrations={globalConfig.integrations}
            availablePolicies={policiesAvailableToAutomate}
            isUpdating={isUpdatingPolicies}
            onExit={toggleOtherWorkflowsModal}
            onSubmit={onUpdateOtherWorkflows}
            teamId={currentTeamId ?? 0}
            gitOpsModeEnabled={gitOpsModeEnabled}
          />
        )}
        {showDeletePoliciesModal && (
          <DeletePoliciesModal
            isUpdatingPolicies={isUpdatingPolicies}
            onCancel={toggleDeletePoliciesModal}
            onSubmit={onDeletePolicySubmit}
          />
        )}
        {showInstallSoftwareModal && (
          <InstallSoftwareModal
            onExit={toggleInstallSoftwareModal}
            onSubmit={onUpdatePolicySoftwareInstall}
            isUpdating={isUpdatingPolicies}
            // currentTeamId will at this point be present
            teamId={currentTeamId ?? 0}
            gitOpsModeEnabled={gitOpsModeEnabled}
          />
        )}
        {showPolicyRunScriptModal && (
          <PolicyRunScriptModal
            onExit={togglePolicyRunScriptModal}
            onSubmit={onUpdatePolicyRunScript}
            isUpdating={isUpdatingPolicies}
            // currentTeamId will at this point be present
            teamId={currentTeamId ?? 0}
          />
        )}
        {showCalendarEventsModal && (
          <CalendarEventsModal
            onExit={toggleCalendarEventsModal}
            onSubmit={onUpdateCalendarEvents}
            configured={isCalEventsConfigured}
            enabled={isCalEventsEnabled}
            url={teamConfig?.integrations.google_calendar?.webhook_url || ""}
            teamId={currentTeamId ?? 0}
            isUpdating={isUpdatingPolicies}
            gitOpsModeEnabled={gitOpsModeEnabled}
          />
        )}
        {showConditionalAccessModal && (
          <ConditionalAccessModal
            onExit={toggleConditionalAccessModal}
            onSubmit={onUpdateConditionalAccess}
            configured={isConditionalAccessConfigured}
            enabled={isConditionalAccessEnabled}
            isUpdating={isUpdatingPolicies}
            gitOpsModeEnabled={gitOpsModeEnabled}
            teamId={currentTeamId ?? 0}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManagePolicyPage;
