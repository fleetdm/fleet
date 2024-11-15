import React, { useState, useEffect, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import {
  DEFAULT_QUERY,
  DOCUMENT_TITLE_SUFFIX,
  INVALID_PLATFORMS_FLASH_MESSAGE,
  INVALID_PLATFORMS_REASON,
} from "utilities/constants";
import configAPI from "services/entities/config";
import queryAPI from "services/entities/queries";
import statusAPI from "services/entities/status";
import {
  IGetQueryResponse,
  ICreateQueryRequestBody,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { IConfig } from "interfaces/config";
import { getErrorReason } from "interfaces/errors";

import QuerySidePanel from "components/side_panels/QuerySidePanel";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import CustomLink from "components/CustomLink";
import BackLink from "components/BackLink";
import InfoBanner from "components/InfoBanner";

import useTeamIdParam from "hooks/useTeamIdParam";

import { NotificationContext } from "context/notification";

import PATHS from "router/paths";
import debounce from "utilities/debounce";
import deepDifference from "utilities/deep_difference";
import { buildQueryStringFromParams } from "utilities/url";

import EditQueryForm from "./components/EditQueryForm";

interface IEditQueryPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    query: { host_id: string; team_id?: string };
    search: string;
  };
}

const baseClass = "edit-query-page";

const EditQueryPage = ({
  router,
  params: { id: paramsQueryId },
  location,
}: IEditQueryPageProps): JSX.Element => {
  const queryId = paramsQueryId ? parseInt(paramsQueryId, 10) : null;

  const {
    currentTeamName: teamNameForQuery,
    teamIdForApi: apiTeamIdForQuery,
    currentTeamId,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const handlePageError = useErrorHandler();
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamMaintainerOrTeamAdmin,
    isAnyTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    isAnyTeamObserverPlus,
    config,
    filteredQueriesPath,
    isOnGlobalTeam,
  } = useContext(AppContext);
  const {
    editingExistingQuery,
    selectedOsqueryTable,
    setSelectedOsqueryTable,
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryObserverCanRun,
    lastEditedQueryFrequency,
    lastEditedQueryAutomationsEnabled,
    lastEditedQueryPlatforms,
    lastEditedQueryLoggingType,
    lastEditedQueryMinOsqueryVersion,
    lastEditedQueryDiscardData,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
    setLastEditedQueryFrequency,
    setLastEditedQueryAutomationsEnabled,
    setLastEditedQueryLoggingType,
    setLastEditedQueryMinOsqueryVersion,
    setLastEditedQueryPlatforms,
    setLastEditedQueryDiscardData,
  } = useContext(QueryContext);
  const { setConfig, availableTeams, setCurrentTeam } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [isLiveQueryRunnable, setIsLiveQueryRunnable] = useState(true);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [showOpenSchemaActionText, setShowOpenSchemaActionText] = useState(
    false
  );
  const [
    showConfirmSaveChangesModal,
    setShowConfirmSaveChangesModal,
  ] = useState(false);

  const { data: appConfig } = useQuery<IConfig, Error, IConfig>(
    ["config"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfig) => data,
      onSuccess: (data) => {
        setConfig(data);
      },
    }
  );

  // disabled on page load so we can control the number of renders
  // else it will re-populate the context on occasion
  const {
    isLoading: isStoredQueryLoading,
    data: storedQuery,
    refetch: refetchStoredQuery,
  } = useQuery<IGetQueryResponse, Error, ISchedulableQuery>(
    ["query", queryId],
    () => queryAPI.load(queryId as number),
    {
      enabled: !!queryId && !editingExistingQuery,
      refetchOnWindowFocus: false,
      select: (data) => data.query,
      onSuccess: (returnedQuery) => {
        setLastEditedQueryId(returnedQuery.id);
        setLastEditedQueryName(returnedQuery.name);
        setLastEditedQueryDescription(returnedQuery.description);
        setLastEditedQueryBody(returnedQuery.query);
        setLastEditedQueryObserverCanRun(returnedQuery.observer_can_run);
        setLastEditedQueryFrequency(returnedQuery.interval);
        setLastEditedQueryAutomationsEnabled(returnedQuery.automations_enabled);
        setLastEditedQueryPlatforms(returnedQuery.platform);
        setLastEditedQueryLoggingType(returnedQuery.logging);
        setLastEditedQueryMinOsqueryVersion(returnedQuery.min_osquery_version);
        setLastEditedQueryDiscardData(returnedQuery.discard_data);
      },
      onError: (error) => handlePageError(error),
    }
  );

  /** Pesky bug affecting team level users:
   - Navigating to queries/:id immediately defaults the user to the first team they're on
  with the most permissions, in the URL bar because of useTeamIdParam
  even if the queries/:id entity has a team attached to it
  Hacky fix:
   - Push entity's team id to url for team level users
  */
  if (
    !isOnGlobalTeam &&
    !isStoredQueryLoading &&
    storedQuery?.team_id &&
    !(storedQuery?.team_id?.toString() === location.query.team_id)
  ) {
    router.push(
      `${location.pathname}?team_id=${storedQuery?.team_id?.toString()}`
    );
  }

  // Used to set host's team in AppContext for RBAC actions
  useEffect(() => {
    if (storedQuery?.team_id) {
      const querysTeam = availableTeams?.find(
        (team) => team.id === storedQuery.team_id
      );
      setCurrentTeam(querysTeam);
    }
  }, [storedQuery]);

  const detectIsFleetQueryRunnable = () => {
    statusAPI.live_query().catch(() => {
      setIsLiveQueryRunnable(false);
    });
  };

  /* Observer/Observer+ cannot edit existing query (O+ has access to edit new query to run live),
  Team admin/team maintainer cannot edit existing query,
 reroute edit existing query page (/:queryId/edit) to query report page (/:queryId) */
  useEffect(() => {
    const canEditExistingQuery =
      isGlobalAdmin ||
      isGlobalMaintainer ||
      (isTeamMaintainerOrTeamAdmin && storedQuery?.team_id);

    if (
      !isStoredQueryLoading && // Confirms teamId for storedQuery before RBAC reroute
      queryId &&
      queryId > 0 &&
      !canEditExistingQuery
    ) {
      // Reroute to query report page still maintains query params for live query purposes
      const baseUrl = PATHS.QUERY_DETAILS(queryId);
      const queryParams = buildQueryStringFromParams({
        host_id: location.query.host_id,
        team_id: location.query.team_id,
      });

      router.push(queryParams ? `${baseUrl}?${queryParams}` : baseUrl);
    }
  }, [queryId, isTeamMaintainerOrTeamAdmin, isStoredQueryLoading]);

  useEffect(() => {
    detectIsFleetQueryRunnable();
    if (!queryId) {
      setLastEditedQueryId(DEFAULT_QUERY.id);
      setLastEditedQueryName(DEFAULT_QUERY.name);
      setLastEditedQueryDescription(DEFAULT_QUERY.description);
      // Persist lastEditedQueryBody through live query flow instead of resetting to DEFAULT_QUERY.query
      setLastEditedQueryObserverCanRun(DEFAULT_QUERY.observer_can_run);
      setLastEditedQueryFrequency(DEFAULT_QUERY.interval);
      setLastEditedQueryAutomationsEnabled(DEFAULT_QUERY.automations_enabled);
      setLastEditedQueryLoggingType(DEFAULT_QUERY.logging);
      setLastEditedQueryMinOsqueryVersion(DEFAULT_QUERY.min_osquery_version);
      setLastEditedQueryPlatforms(DEFAULT_QUERY.platform);
      setLastEditedQueryDiscardData(DEFAULT_QUERY.discard_data);
    }
  }, [queryId]);

  const [isQuerySaving, setIsQuerySaving] = useState(false);
  const [isQueryUpdating, setIsQueryUpdating] = useState(false);
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Editing Discover TLS certificates | Queries | Fleet
    const storedQueryTitleCopy = storedQuery?.name
      ? `Editing ${storedQuery.name} | `
      : "";
    document.title = `${storedQueryTitleCopy}Queries | ${DOCUMENT_TITLE_SUFFIX}`;
    // }
  }, [location.pathname, storedQuery?.name]);

  useEffect(() => {
    setShowOpenSchemaActionText(!isSidebarOpen);
  }, [isSidebarOpen]);

  const onSubmitNewQuery = debounce(
    async (formData: ICreateQueryRequestBody) => {
      setIsQuerySaving(true);
      try {
        const { query } = await queryAPI.create(formData);
        router.push(PATHS.QUERY_DETAILS(query.id, query.team_id));
        renderFlash("success", "Query created!");
        setBackendValidators({});
      } catch (createError: any) {
        if (getErrorReason(createError).includes("already exists")) {
          const teamErrorText =
            teamNameForQuery && apiTeamIdForQuery !== 0
              ? `the ${teamNameForQuery} team`
              : "all teams";
          setBackendValidators({
            name: `A query with that name already exists for ${teamErrorText}.`,
          });
        } else {
          renderFlash(
            "error",
            "Something went wrong creating your query. Please try again."
          );
          setBackendValidators({});
        }
      } finally {
        setIsQuerySaving(false);
      }
    }
  );

  const onUpdateQuery = async (formData: ICreateQueryRequestBody) => {
    if (!queryId) {
      return false;
    }

    setIsQueryUpdating(true);

    const updatedQuery = deepDifference(formData, {
      lastEditedQueryName,
      lastEditedQueryDescription,
      lastEditedQueryBody,
      lastEditedQueryObserverCanRun,
      lastEditedQueryFrequency,
      lastEditedQueryAutomationsEnabled,
      lastEditedQueryPlatforms,
      lastEditedQueryLoggingType,
      lastEditedQueryMinOsqueryVersion,
      lastEditedQueryDiscardData,
    });

    try {
      await queryAPI.update(queryId, updatedQuery);
      renderFlash("success", "Query updated!");
      refetchStoredQuery(); // Required to compare recently saved query to a subsequent save to the query
    } catch (updateError: any) {
      console.error(updateError);
      const reason = getErrorReason(updateError);
      if (reason.includes("Duplicate")) {
        renderFlash("error", "A query with this name already exists.");
      } else if (reason.includes(INVALID_PLATFORMS_REASON)) {
        renderFlash("error", INVALID_PLATFORMS_FLASH_MESSAGE);
      } else {
        renderFlash(
          "error",
          "Something went wrong updating your query. Please try again."
        );
      }
    }

    setIsQueryUpdating(false);
    setShowConfirmSaveChangesModal(false); // Closes conditionally opened modal when discarding previous results

    return false;
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const onCloseSchemaSidebar = () => {
    setIsSidebarOpen(false);
  };

  const onOpenSchemaSidebar = () => {
    setIsSidebarOpen(true);
  };

  const renderLiveQueryWarning = (): JSX.Element | null => {
    if (isLiveQueryRunnable || config?.server_settings.live_query_disabled) {
      return null;
    }

    return (
      <InfoBanner color="yellow">
        Fleet is unable to run a live query. Refresh the page or log in again.
        If this keeps happening please{" "}
        <CustomLink
          url="https://github.com/fleetdm/fleet/issues/new/choose"
          text="file an issue"
          newTab
        />
      </InfoBanner>
    );
  };

  // Function instead of constant eliminates race condition
  const backToQueriesPath = () => {
    const manageQueryPage =
      filteredQueriesPath ||
      `${PATHS.MANAGE_QUERIES}?${buildQueryStringFromParams({
        team_id: currentTeamId,
      })}`;

    return queryId
      ? PATHS.QUERY_DETAILS(queryId, currentTeamId)
      : manageQueryPage;
  };

  const showSidebar =
    isSidebarOpen &&
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isAnyTeamMaintainerOrTeamAdmin ||
      isObserverPlus ||
      isAnyTeamObserverPlus);

  return (
    <>
      <MainContent className={baseClass}>
        <>
          <div className={`${baseClass}__header-links`}>
            <BackLink
              text={queryId ? "Back to report" : "Back to queries"}
              path={backToQueriesPath()}
            />
          </div>
          <EditQueryForm
            router={router}
            onSubmitNewQuery={onSubmitNewQuery}
            onOsqueryTableSelect={onOsqueryTableSelect}
            onUpdate={onUpdateQuery}
            storedQuery={storedQuery}
            queryIdForEdit={queryId}
            apiTeamIdForQuery={apiTeamIdForQuery}
            currentTeamId={currentTeamId}
            teamNameForQuery={teamNameForQuery}
            isStoredQueryLoading={isStoredQueryLoading}
            showOpenSchemaActionText={showOpenSchemaActionText}
            onOpenSchemaSidebar={onOpenSchemaSidebar}
            renderLiveQueryWarning={renderLiveQueryWarning}
            backendValidators={backendValidators}
            isQuerySaving={isQuerySaving}
            isQueryUpdating={isQueryUpdating}
            hostId={parseInt(location.query.host_id as string, 10)}
            queryReportsDisabled={
              appConfig?.server_settings.query_reports_disabled
            }
            showConfirmSaveChangesModal={showConfirmSaveChangesModal}
            setShowConfirmSaveChangesModal={setShowConfirmSaveChangesModal}
          />
        </>
      </MainContent>
      {showSidebar && (
        <SidePanelContent>
          <QuerySidePanel
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={onCloseSchemaSidebar}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default EditQueryPage;
