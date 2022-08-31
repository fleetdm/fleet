import React, {
  useContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { pick } from "lodash";

import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import { performanceIndicator } from "utilities/helpers";
import { IOsqueryPlatform } from "interfaces/platform";
import { IQuery } from "interfaces/query";
import fleetQueriesAPI from "services/entities/queries";
import PATHS from "router/paths";
import checkPlatformCompatibility from "utilities/sql_tools";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import QueriesTable from "./components/QueriesTable";
import DeleteQueryModal from "./components/DeleteQueryModal";

const baseClass = "manage-queries-page";
interface IManageQueriesPageProps {
  router: InjectedRouter; // v3
}

interface IFleetQueriesResponse {
  queries: IQuery[];
}
interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}

const PLATFORM_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "All platforms",
    value: "all",
    helpText: "All queries.",
  },
  {
    disabled: false,
    label: "Linux",
    value: "linux",
    helpText: "Queries that are compatible with Linux operating systems.",
  },
  {
    disabled: false,
    label: "macOS",
    value: "darwin",
    helpText: "Queries that are compatible with macOS operating systems.",
  },
  {
    disabled: false,
    label: "Windows",
    value: "windows",
    helpText: "Queries that are compatible with Windows operating systems.",
  },
];

const getPlatforms = (queryString: string): Array<IOsqueryPlatform | "---"> => {
  const { platforms } = checkPlatformCompatibility(queryString);

  return platforms || ["---"];
};

const enhanceQuery = (q: IQuery) => {
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
}: IManageQueriesPageProps): JSX.Element => {
  const { isOnlyObserver } = useContext(AppContext);
  const { setResetSelectedRows } = useContext(TableContext);
  const { renderFlash } = useContext(NotificationContext);

  const [queriesList, setQueriesList] = useState<IQueryTableData[] | null>(
    null
  );
  const [selectedDropdownFilter, setSelectedDropdownFilter] = useState<string>(
    "all"
  );
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showDeleteQueryModal, setShowDeleteQueryModal] = useState<boolean>(
    false
  );
  const [isUpdatingQueries, setIsUpdatingQueries] = useState<boolean>(false);

  const {
    data: fleetQueries,
    error: fleetQueriesError,
    isFetching: isFetchingFleetQueries,
    refetch: refetchFleetQueries,
  } = useQuery<IFleetQueriesResponse, Error, IQuery[]>(
    "fleet queries by platform",
    () => fleetQueriesAPI.loadAll(),
    {
      refetchOnWindowFocus: false,
      select: (data: IFleetQueriesResponse) => data.queries,
    }
  );

  const enhancedQueriesList = useMemo(() => {
    const enhancedQueries = fleetQueries?.map((q: IQuery) => {
      const query = enhanceQuery(q);
      return query;
    });

    return enhancedQueries || [];
  }, [fleetQueries]);

  useEffect(() => {
    if (!isFetchingFleetQueries && enhancedQueriesList) {
      setQueriesList(enhancedQueriesList);
    }
  }, [enhancedQueriesList, isFetchingFleetQueries]);

  const onCreateQueryClick = () => router.push(PATHS.NEW_QUERY);

  const toggleDeleteQueryModal = useCallback(() => {
    setShowDeleteQueryModal(!showDeleteQueryModal);
  }, [showDeleteQueryModal, setShowDeleteQueryModal]);

  const onDeleteQueryClick = (selectedTableQueryIds: number[]) => {
    toggleDeleteQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onDeleteQuerySubmit = useCallback(async () => {
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";

    setIsUpdatingQueries(true);

    const deleteQueries = selectedQueryIds.map((id) =>
      fleetQueriesAPI.destroy(id)
    );

    try {
      await Promise.all(deleteQueries).then(() => {
        renderFlash("success", `Successfully deleted ${queryOrQueries}.`);
        setResetSelectedRows(true);
        refetchFleetQueries();
      });
      renderFlash("success", `Successfully deleted ${queryOrQueries}.`);
    } catch (errorResponse) {
      renderFlash(
        "error",
        `There was an error deleting your ${queryOrQueries}. Please try again later.`
      );
    } finally {
      toggleDeleteQueryModal();
      setIsUpdatingQueries(false);
    }
  }, [refetchFleetQueries, selectedQueryIds, toggleDeleteQueryModal]);

  const renderPlatformDropdown = () => {
    return (
      <Dropdown
        value={selectedDropdownFilter}
        className={`${baseClass}__platform_dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        searchable={false}
        onChange={setSelectedDropdownFilter}
      />
    );
  };

  const isTableDataLoading = isFetchingFleetQueries || queriesList === null;

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Queries</span>
              </h1>
            </div>
          </div>
          {!isOnlyObserver && !!fleetQueries?.length && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                variant="brand"
                className={`${baseClass}__create-button`}
                onClick={onCreateQueryClick}
              >
                Create new query
              </Button>
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          <p>Manage queries to ask specific questions about your devices.</p>
        </div>
        <div>
          {isTableDataLoading && !fleetQueriesError && <Spinner />}
          {!isTableDataLoading && fleetQueriesError ? (
            <TableDataError />
          ) : (
            <QueriesTable
              queriesList={queriesList}
              isLoading={isTableDataLoading}
              onCreateQueryClick={onCreateQueryClick}
              onDeleteQueryClick={onDeleteQueryClick}
              customControl={renderPlatformDropdown}
              selectedDropdownFilter={selectedDropdownFilter}
              isOnlyObserver={isOnlyObserver}
            />
          )}
        </div>
        {showDeleteQueryModal && (
          <DeleteQueryModal
            isUpdatingQueries={isUpdatingQueries}
            onCancel={toggleDeleteQueryModal}
            onSubmit={onDeleteQuerySubmit}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManageQueriesPage;
