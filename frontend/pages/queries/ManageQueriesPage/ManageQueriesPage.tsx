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
import { API_ALL_TEAMS_ID } from "interfaces/team";
import {
  IEnhancedQuery,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { DEFAULT_TARGETS_BY_TYPE } from "interfaces/target";
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
import RevealButton from "components/buttons/RevealButton";
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
      inherited_order_key?: string;
      inherited_order_direction?: "asc" | "desc";
      inherited_page?: string;
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
  const [showInheritedQueries, setShowInheritedQueries] = useState(false);
  const [isUpdatingAutomations, setIsUpdatingAutomations] = useState(false);

  const {
    data: curTeamEnhancedQueries,
    error: curTeamQueriesError,
    isFetching: isFetchingCurTeamQueries,
    refetch: refetchCurTeamQueries,
  } = useQuery<
    IEnhancedQuery[],
    Error,
    IEnhancedQuery[],
    IQueryKeyQueriesLoadAll[]
  >(
    [{ scope: "queries", teamId: teamIdForApi }],
    ({ queryKey: [{ teamId }] }) =>
      queriesAPI
        .loadAll(teamId)
        .then(({ queries }) => queries.map(enhanceQuery)),
    {
      refetchOnWindowFocus: false,
      enabled: isRouteOk,
      staleTime: 5000,
    }
  );

  // If a team is selected, inherit global queries
  const {
    data: globalEnhancedQueries,
    error: globalQueriesError,
    isFetching: isFetchingGlobalQueries,
    refetch: refetchGlobalQueries,
  } = useQuery<
    IEnhancedQuery[],
    Error,
    IEnhancedQuery[],
    IQueryKeyQueriesLoadAll[]
  >(
    [{ scope: "queries", teamId: API_ALL_TEAMS_ID }],
    ({ queryKey: [{ teamId }] }) =>
      queriesAPI
        .loadAll(teamId)
        .then(({ queries }) => queries.map(enhanceQuery)),
    {
      refetchOnWindowFocus: false,
      enabled: isRouteOk && isAnyTeamSelected,
      staleTime: 5000,
    }
  );

  const automatedQueryIds = useMemo(() => {
    return curTeamEnhancedQueries
      ? curTeamEnhancedQueries
          .filter((query) => query.automations_enabled)
          .map((query) => query.id)
      : [];
  }, [curTeamEnhancedQueries]);

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

  const refetchAllQueries = useCallback(() => {
    refetchCurTeamQueries();
    refetchGlobalQueries();
  }, [refetchCurTeamQueries, refetchGlobalQueries]);

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
      refetchAllQueries();
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
  }, [refetchAllQueries, selectedQueryIds, toggleDeleteQueryModal]);

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

  const renderCurrentScopeQueriesTable = () => {
    if (isFetchingCurTeamQueries) {
      return <Spinner />;
    }
    if (curTeamQueriesError) {
      return <TableDataError />;
    }
    return (
      <QueriesTable
        queriesList={curTeamEnhancedQueries || []}
        isLoading={isFetchingCurTeamQueries}
        onCreateQueryClick={onCreateQueryClick}
        onDeleteQueryClick={onDeleteQueryClick}
        isOnlyObserver={isOnlyObserver}
        isObserverPlus={isObserverPlus}
        isAnyTeamObserverPlus={isAnyTeamObserverPlus || false}
        router={router}
        queryParams={queryParams}
      />
    );
  };

  const renderShowInheritedQueriesTableButton = () => {
    const inheritedQueryCount = globalEnhancedQueries?.length;
    return (
      <RevealButton
        isShowing={showInheritedQueries}
        className={baseClass}
        hideText={`Hide ${inheritedQueryCount} inherited quer${
          inheritedQueryCount === 1 ? "y" : "ies"
        }`}
        showText={`Show ${inheritedQueryCount} inherited quer${
          inheritedQueryCount === 1 ? "y" : "ies"
        }`}
        caretPosition="before"
        tooltipContent={
          <>
            Queries from the &quot;All teams&quot;
            <br />
            schedule run on this team&apos;s hosts.
          </>
        }
        onClick={() => {
          setShowInheritedQueries(!showInheritedQueries);
        }}
      />
    );
  };

  const renderInheritedQueriesTable = () => {
    if (isFetchingGlobalQueries) {
      return <Spinner />;
    }
    if (globalQueriesError) {
      return <TableDataError />;
    }
    return (
      <QueriesTable
        queriesList={globalEnhancedQueries || []}
        isLoading={isFetchingGlobalQueries}
        onCreateQueryClick={onCreateQueryClick}
        onDeleteQueryClick={onDeleteQueryClick}
        isOnlyObserver={isOnlyObserver}
        isObserverPlus={isObserverPlus}
        isAnyTeamObserverPlus={isAnyTeamObserverPlus || false}
        router={router}
        queryParams={queryParams}
        isInherited
        currentTeamId={currentTeamId}
      />
    );
  };

  const renderInheritedQueriesSection = () => {
    return (
      <>
        {renderShowInheritedQueriesTableButton()}
        {showInheritedQueries && renderInheritedQueriesTable()}
      </>
    );
  };

  const onSaveQueryAutomations = useCallback(
    async (newAutomatedQueryIds) => {
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
          refetchAllQueries();
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
    [refetchAllQueries, automatedQueryIds, toggleManageAutomationsModal]
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
            availableQueries={curTeamEnhancedQueries}
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
          <div className={`${baseClass}__action-button-container`}>
            {(isGlobalAdmin || isTeamAdmin) && (
              <Button
                onClick={onManageAutomationsClick}
                className={`${baseClass}__manage-automations button`}
                variant="inverse"
              >
                Manage automations
              </Button>
            )}
            {(!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) &&
              !!curTeamEnhancedQueries?.length && (
                <>
                  <Button
                    variant="brand"
                    className={`${baseClass}__create-button`}
                    onClick={onCreateQueryClick}
                  >
                    {isObserverPlus ? "Live query" : "Add query"}
                  </Button>
                </>
              )}
          </div>
        </div>
        <div className={`${baseClass}__description`}>
          <p>
            {isAnyTeamSelected
              ? "Gather data about all hosts assigned to this team."
              : "Gather data about all hosts."}
          </p>
        </div>
        {renderCurrentScopeQueriesTable()}
        {isAnyTeamSelected &&
          globalEnhancedQueries &&
          globalEnhancedQueries?.length > 0 &&
          renderInheritedQueriesSection()}
        {renderModals()}
      </div>
    </MainContent>
  );
};

export default ManageQueriesPage;
