import React, { useContext, useCallback, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { pick } from "lodash";

import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import { performanceIndicator } from "utilities/helpers";
import { OsqueryPlatform } from "interfaces/platform";
import {
  IListQueriesResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import queriesAPI from "services/entities/queries";
import PATHS from "router/paths";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import checkPlatformCompatibility from "utilities/sql_tools";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";
import useTeamIdParam from "hooks/useTeamIdParam";
import RevealButton from "components/buttons/RevealButton";
import QueriesTable from "./components/QueriesTable";
import DeleteQueryModal from "./components/DeleteQueryModal";
import ManageAutomationsModal from "./components/ManageAutomationsModal";

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

interface IEnhancedQuery extends ISchedulableQuery {
  performance: string;
  platforms: string[];
}

const getPlatforms = (queryString: string): Array<OsqueryPlatform | "---"> => {
  const { platforms } = checkPlatformCompatibility(queryString);

  return platforms || [DEFAULT_EMPTY_CELL_VALUE];
};

const enhanceQuery = (q: ISchedulableQuery): IEnhancedQuery => {
  return {
    ...q,
    performance: performanceIndicator(
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
  } = useContext(AppContext);

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
  const [isUpdatingQueries, setIsUpdatingQueries] = useState(false);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showInheritedQueries, setShowInheritedQueries] = useState(false);

  const {
    data: curTeamEnhancedQueries,
    error: curTeamQueriesError,
    isFetching: isFetchingCurTeamQueries,
    refetch: refetchCurTeamQueries,
  } = useQuery<IListQueriesResponse, Error, IEnhancedQuery[]>(
    [{ scope: "queries", teamId: teamIdForApi }],
    () => queriesAPI.loadAll(teamIdForApi),
    {
      refetchOnWindowFocus: false,
      enabled: isRouteOk,
      select: (data) => data.queries.map(enhanceQuery),
    }
  );

  // If a team is selected, fetch inherited global queries as well
  const {
    data: globalEnhancedQueries,
    error: globalQueriesError,
    isFetching: isFetchingGlobalQueries,
    refetch: refetchGlobalQueries,
  } = useQuery<IListQueriesResponse, Error, IEnhancedQuery[]>(
    [{ scope: "queries", teamId: -1 }],
    () => queriesAPI.loadAll(),
    {
      refetchOnWindowFocus: false,
      enabled: isRouteOk && isAnyTeamSelected,
      select: (data) => data.queries.map(enhanceQuery),
    }
  );

  useEffect(() => {
    const path = location.pathname + location.search;
    if (filteredQueriesPath !== path) {
      setFilteredQueriesPath(path);
    }
  }, [location, filteredQueriesPath, setFilteredQueriesPath]);

  const onCreateQueryClick = () => router.push(PATHS.NEW_QUERY);

  const toggleDeleteQueryModal = useCallback(() => {
    setShowDeleteQueryModal(!showDeleteQueryModal);
  }, [showDeleteQueryModal, setShowDeleteQueryModal]);

  const toggleManageAutomationsModal = useCallback(() => {
    setShowManageAutomationsModal(!showManageAutomationsModal);
  }, [showManageAutomationsModal, setShowManageAutomationsModal]);

  const onDeleteQueryClick = (selectedTableQueryIds: number[]) => {
    toggleDeleteQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const refetchAllQueries = useCallback(() => {
    refetchCurTeamQueries();
    refetchGlobalQueries();
  }, [refetchCurTeamQueries, refetchGlobalQueries]);

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
      <div>
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
      </div>
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
        caretPosition={"before"}
        tooltipHtml={
          'Queries from the "All teams"<br/>schedule run on this team’s hosts.'
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
      <div>
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
        />
      </div>
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
          <ManageAutomationsModal onExit={toggleManageAutomationsModal} />
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
                onClick={toggleManageAutomationsModal}
                className={`${baseClass}__manage-automations button`}
                variant="inverse"
              >
                Manage automations
              </Button>
            )}
            {(!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) &&
              !!curTeamEnhancedQueries?.length && (
                <Button
                  variant="brand"
                  className={`${baseClass}__create-button`}
                  onClick={onCreateQueryClick}
                >
                  Add query
                </Button>
              )}
          </div>
        </div>
        <div className={`${baseClass}__description`}>
          <p>
            Manage and schedule queries to ask questions and collect telemetry
            for all hosts{isAnyTeamSelected && " assigned to this team"}.
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
