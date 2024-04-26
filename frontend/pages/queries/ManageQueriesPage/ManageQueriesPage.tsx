import React, {
  useContext,
  useCallback,
  useEffect,
  useState,
  useMemo,
} from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { pick } from "lodash";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import { getPerformanceImpactDescription } from "utilities/helpers";
import { SupportedPlatform } from "interfaces/platform";
import {
  IEnhancedQuery,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { DEFAULT_TARGETS_BY_TYPE } from "interfaces/target";
import { API_ALL_TEAMS_ID } from "interfaces/team";
import queriesAPI from "services/entities/queries";
import PATHS from "router/paths";
import { DEFAULT_QUERY } from "utilities/constants";
import { checkPlatformCompatibility } from "utilities/sql_tools";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";
import useTeamIdParam from "hooks/useTeamIdParam";
import QueriesTable from "./components/QueriesTable";
import DeleteQueryModal from "./components/DeleteQueryModal";
import ManageQueryAutomationsModal from "./components/ManageQueryAutomationsModal/ManageQueryAutomationsModal";
import PreviewDataModal from "./components/PreviewDataModal/PreviewDataModal";

const baseClass = "manage-queries-page";
interface IManageQueriesPageProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: {
      platform?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      team_id?: string;
    };
    search: string;
  };
}

const getPlatforms = (queryString: string): SupportedPlatform[] => {
  const { platforms } = checkPlatformCompatibility(queryString);

  return platforms ?? [];
};

const enhanceQuery = (q: ISchedulableQuery): IEnhancedQuery => {
  return {
    ...q,
    performance: getPerformanceImpactDescription(
      pick(q.stats, ["user_time_p50", "system_time_p50", "total_executions"])
    ),
    platforms: getPlatforms(q.query),
  };
};

const ManageQueriesPage = ({
  router,
  location,
}: IManageQueriesPageProps): JSX.Element => {
  const queryParams = location.query;
  const {
    isGlobalAdmin,
    isTeamAdmin,
    isOnlyObserver,
    isObserverPlus,
    isAnyTeamObserverPlus,
    isOnGlobalTeam,
    setFilteredQueriesPath,
    filteredQueriesPath,
    isPremiumTier,
    isSandboxMode,
    config,
  } = useContext(AppContext);
  const { setLastEditedQueryBody, setSelectedQueryTargetsByType } = useContext(
    QueryContext
  );
  const { setResetSelectedRows } = useContext(TableContext);
  const { renderFlash } = useContext(NotificationContext);

  const {
    userTeams,
    currentTeamId,
    handleTeamChange,
    teamIdForApi,
    isRouteOk,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const isAnyTeamSelected = currentTeamId !== -1;

  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showDeleteQueryModal, setShowDeleteQueryModal] = useState(false);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewDataModal, setShowPreviewDataModal] = useState(false);
  const [isUpdatingQueries, setIsUpdatingQueries] = useState(false);
  const [isUpdatingAutomations, setIsUpdatingAutomations] = useState(false);

  // const {
  //   data: enhancedQueries,
  //   error: queriesError,
  //   isFetching: isFetchingQueries,
  //   refetch: refetchQueries,
  // } = useQuery<
  //   IEnhancedQuery[],
  //   Error,
  //   IEnhancedQuery[],
  //   IQueryKeyQueriesLoadAll[]
  // >(
  //   [{ scope: "queries", teamId: teamIdForApi }],
  //   ({ queryKey: [{ teamId }] }) =>
  //     queriesAPI
  //       .loadAll(teamId, teamId !== API_ALL_TEAMS_ID)
  //       .then(({ queries }) => queries.map(enhanceQuery)),
  //   {
  //     refetchOnWindowFocus: false,
  //     enabled: isRouteOk,
  //     staleTime: 5000,
  //   }
  // );

  // TODO - restore API call
  const rawTestqueries: ISchedulableQuery[] = [
    // global query
    {
      created_at: "2024-03-22T19:01:20Z",
      updated_at: "2024-03-22T19:01:20Z",
      id: 5,
      team_id: null,
      interval: 0,
      platform: "linux",
      min_osquery_version: "",
      automations_enabled: false,
      logging: "snapshot",
      name: "Get OpenSSL versions",
      description: "Retrieves the OpenSSL version.",
      query:
        "SELECT name AS name, version AS version, 'deb_packages' AS source FROM deb_packages WHERE name LIKE 'openssl%' UNION SELECT name AS name, version AS version, 'apt_sources' AS source FROM apt_sources WHERE name LIKE 'openssl%' UNION SELECT name AS name, version AS version, 'rpm_packages' AS source FROM rpm_packages WHERE name LIKE 'openssl%';",
      saved: true,
      observer_can_run: false,
      author_id: 1,
      author_name: "J Cob",
      author_email: "jacob@fleetdm.com",
      packs: [],
      stats: {
        system_time_p50: null,
        system_time_p95: null,
        user_time_p50: null,
        user_time_p95: null,
        total_executions: 0,
      },
      discard_data: false,
    },
    // team queries
    {
      created_at: "2024-04-25T04:16:09Z",
      updated_at: "2024-04-25T04:16:09Z",
      id: 94,
      team_id: 2,
      // team_id: null,
      interval: 3600,
      platform: "",
      min_osquery_version: "",
      automations_enabled: false,
      logging: "snapshot",
      name: "Oranges 2",
      description: "",
      query: "SELECT * FROM osquery_info;",
      saved: true,
      observer_can_run: false,
      author_id: 1,
      author_name: "J Cob",
      author_email: "jacob@fleetdm.com",
      packs: [],
      stats: {
        system_time_p50: null,
        system_time_p95: null,
        user_time_p50: null,
        user_time_p95: null,
        total_executions: 0,
      },
      discard_data: false,
    },
    {
      created_at: "2024-04-25T04:16:09Z",
      updated_at: "2024-04-25T04:16:09Z",
      id: 93,
      team_id: 2,
      // team_id: null,
      interval: 3600,
      platform: "",
      min_osquery_version: "",
      automations_enabled: false,
      logging: "snapshot",
      name: "Oranges 1",
      description: "",
      query: "SELECT * FROM osquery_info;",
      saved: true,
      observer_can_run: false,
      author_id: 1,
      author_name: "J Cob",
      author_email: "jacob@fleetdm.com",
      packs: [],
      stats: {
        system_time_p50: null,
        system_time_p95: null,
        user_time_p50: null,
        user_time_p95: null,
        total_executions: 0,
      },
      discard_data: false,
    },
  ];
  const [queriesError, refetchQueries, isFetchingQueries, enhancedQueries] = [
    null,
    () => undefined,
    false,
    rawTestqueries.map(enhanceQuery),
    // [] as IEnhancedQuery[],
  ];

  const onlyInheritedQueries = useMemo(() => {
    if (teamIdForApi === API_ALL_TEAMS_ID) {
      // global scope
      return false;
    }
    return !enhancedQueries?.some((query) => query.team_id === teamIdForApi);
  }, [teamIdForApi, enhancedQueries]);

  const automatedQueryIds = useMemo(() => {
    return enhancedQueries
      ? enhancedQueries
          .filter((query) => query.automations_enabled)
          .map((query) => query.id)
      : [];
  }, [enhancedQueries]);

  useEffect(() => {
    const path = location.pathname + location.search;
    if (filteredQueriesPath !== path) {
      setFilteredQueriesPath(path);
    }
  }, [location, filteredQueriesPath, setFilteredQueriesPath]);

  // Reset selected targets when returned to this page
  useEffect(() => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
  }, []);

  const onCreateQueryClick = () => {
    setLastEditedQueryBody(DEFAULT_QUERY.query);
    router.push(PATHS.NEW_QUERY(currentTeamId));
  };

  const toggleDeleteQueryModal = useCallback(() => {
    setShowDeleteQueryModal(!showDeleteQueryModal);
  }, [showDeleteQueryModal, setShowDeleteQueryModal]);

  const onDeleteQueryClick = (selectedTableQueryIds: number[]) => {
    toggleDeleteQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const toggleManageAutomationsModal = useCallback(() => {
    setShowManageAutomationsModal(!showManageAutomationsModal);
  }, [showManageAutomationsModal, setShowManageAutomationsModal]);

  const onManageAutomationsClick = () => {
    toggleManageAutomationsModal();
  };

  const togglePreviewDataModal = useCallback(() => {
    // ManageQueryAutomationsModal will be cosmetically hidden when the preview data modal is open
    setShowPreviewDataModal(!showPreviewDataModal);
  }, [showPreviewDataModal, setShowPreviewDataModal]);

  const onDeleteQuerySubmit = useCallback(async () => {
    const bulk = selectedQueryIds.length > 1;
    setIsUpdatingQueries(true);

    try {
      if (bulk) {
        await queriesAPI.bulkDestroy(selectedQueryIds);
      } else {
        await queriesAPI.destroy(selectedQueryIds[0]);
      }
      renderFlash(
        "success",
        `Successfully deleted ${bulk ? "queries" : "query"}.`
      );
      setResetSelectedRows(true);
      refetchQueries();
    } catch (errorResponse) {
      renderFlash(
        "error",
        `There was an error deleting your ${
          bulk ? "queries" : "query"
        }. Please try again later.`
      );
    } finally {
      toggleDeleteQueryModal();
      setIsUpdatingQueries(false);
    }
  }, [refetchQueries, selectedQueryIds, toggleDeleteQueryModal]);

  const renderHeader = () => {
    if (isPremiumTier) {
      if (userTeams) {
        if (userTeams.length > 1 || isOnGlobalTeam) {
          return (
            <TeamsDropdown
              currentUserTeams={userTeams}
              selectedTeamId={currentTeamId}
              onChange={handleTeamChange}
              isSandboxMode={isSandboxMode}
            />
          );
        } else if (!isOnGlobalTeam && userTeams.length === 1) {
          return <h1>{userTeams[0].name}</h1>;
        }
      }
    }
    return <h1>Queries</h1>;
  };

  const renderQueriesTable = () => {
    if (isFetchingQueries) {
      return <Spinner />;
    }
    if (queriesError) {
      return <TableDataError />;
    }
    return (
      <QueriesTable
        queriesList={enhancedQueries || []}
        onlyInheritedQueries={onlyInheritedQueries}
        isLoading={isFetchingQueries}
        onCreateQueryClick={onCreateQueryClick}
        onDeleteQueryClick={onDeleteQueryClick}
        isOnlyObserver={isOnlyObserver}
        isObserverPlus={isObserverPlus}
        isAnyTeamObserverPlus={isAnyTeamObserverPlus || false}
        router={router}
        queryParams={queryParams}
        currentTeamId={teamIdForApi}
      />
    );
  };

  const onSaveQueryAutomations = useCallback(
    async (newAutomatedQueryIds: any) => {
      setIsUpdatingAutomations(true);

      // Query ids added to turn on automations
      const turnOnAutomations = newAutomatedQueryIds.filter(
        (query: number) => !automatedQueryIds.includes(query)
      );
      // Query ids removed to turn off automations
      const turnOffAutomations = automatedQueryIds.filter(
        (query: number) => !newAutomatedQueryIds.includes(query)
      );

      // Update query automations using queries/{id} manage_automations parameter
      const updateAutomatedQueries: Promise<any>[] = [];
      turnOnAutomations.map((id: number) =>
        updateAutomatedQueries.push(
          queriesAPI.update(id, { automations_enabled: true })
        )
      );
      turnOffAutomations.map((id: number) =>
        updateAutomatedQueries.push(
          queriesAPI.update(id, { automations_enabled: false })
        )
      );

      try {
        await Promise.all(updateAutomatedQueries).then(() => {
          renderFlash("success", `Successfully updated query automations.`);
          refetchQueries();
        });
      } catch (errorResponse) {
        renderFlash(
          "error",
          `There was an error updating your query automations. Please try again later.`
        );
      } finally {
        toggleManageAutomationsModal();
        setIsUpdatingAutomations(false);
      }
    },
    [refetchQueries, automatedQueryIds, toggleManageAutomationsModal]
  );

  const renderModals = () => {
    return (
      <>
        {showDeleteQueryModal && (
          <DeleteQueryModal
            isUpdatingQueries={isUpdatingQueries}
            onCancel={toggleDeleteQueryModal}
            onSubmit={onDeleteQuerySubmit}
          />
        )}
        {showManageAutomationsModal && (
          <ManageQueryAutomationsModal
            isUpdatingAutomations={isUpdatingAutomations}
            handleSubmit={onSaveQueryAutomations}
            onCancel={toggleManageAutomationsModal}
            isShowingPreviewDataModal={showPreviewDataModal}
            togglePreviewDataModal={togglePreviewDataModal}
            availableQueries={enhancedQueries}
            automatedQueryIds={automatedQueryIds}
            logDestination={config?.logging.result.plugin || ""}
          />
        )}
        {showPreviewDataModal && (
          <PreviewDataModal onCancel={togglePreviewDataModal} />
        )}
      </>
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>{renderHeader()}</div>
            </div>
          </div>
          {!!enhancedQueries?.length && (
            <div className={`${baseClass}__action-button-container`}>
              {(isGlobalAdmin || isTeamAdmin) && !onlyInheritedQueries && (
                <Button
                  onClick={onManageAutomationsClick}
                  className={`${baseClass}__manage-automations button`}
                  variant="inverse"
                >
                  Manage automations
                </Button>
              )}
              {(!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) && (
                <Button
                  variant="brand"
                  className={`${baseClass}__create-button`}
                  onClick={onCreateQueryClick}
                >
                  {isObserverPlus ? "Live query" : "Add query"}
                </Button>
              )}
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          <p>
            {isAnyTeamSelected
              ? "Gather data about all hosts assigned to this team."
              : "Gather data about all hosts."}
          </p>
        </div>
        {renderQueriesTable()}
        {renderModals()}
      </div>
    </MainContent>
  );
};

export default ManageQueriesPage;
