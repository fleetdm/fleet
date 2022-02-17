import React, {
  useContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { push } from "react-router-redux";
import { pick } from "lodash";

import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { performanceIndicator } from "fleet/helpers";
import { IQuery } from "interfaces/query";
import fleetQueriesAPI from "services/entities/queries";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths";
// @ts-ignore
import sqlTools from "utilities/sql_tools";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TableDataError from "components/TableDataError";
import QueriesListWrapper from "./components/QueriesListWrapper";
import RemoveQueryModal from "./components/RemoveQueryModal";

const baseClass = "manage-queries-page";
interface IFleetQueriesResponse {
  queries: IQuery[];
}
interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}

const PLATFORMS = ["darwin", "linux", "windows"];

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

const getPlatforms = (queryString: string): string[] =>
  sqlTools
    .listCompatiblePlatforms(sqlTools.parseSqlTables(queryString))
    .filter((p: string) => PLATFORMS.includes(p));

const enhanceQuery = (q: IQuery) => {
  return {
    ...q,
    performance: performanceIndicator(
      pick(q.stats, ["user_time_p50", "system_time_p50", "total_executions"])
    ),
    platforms: getPlatforms(q.query),
  };
};

const ManageQueriesPage = (): JSX.Element => {
  const dispatch = useDispatch();

  const { isOnlyObserver } = useContext(AppContext);
  const { setResetSelectedRows } = useContext(TableContext);

  const [queriesList, setQueriesList] = useState<IQueryTableData[] | null>(
    null
  );
  const [selectedDropdownFilter, setSelectedDropdownFilter] = useState<string>(
    "all"
  );
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showRemoveQueryModal, setShowRemoveQueryModal] = useState<boolean>(
    false
  );

  const {
    data: fleetQueries,
    error: fleetQueriesError,
    isLoading: isLoadingFleetQueries,
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
    if (!isLoadingFleetQueries && enhancedQueriesList) {
      setQueriesList(enhancedQueriesList);
    }
  }, [enhancedQueriesList, isLoadingFleetQueries]);

  const onCreateQueryClick = () => dispatch(push(PATHS.NEW_QUERY));

  const toggleRemoveQueryModal = useCallback(() => {
    setShowRemoveQueryModal(!showRemoveQueryModal);
  }, [showRemoveQueryModal, setShowRemoveQueryModal]);

  const onRemoveQueryClick = (selectedTableQueryIds: number[]) => {
    toggleRemoveQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onRemoveQuerySubmit = useCallback(async () => {
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";

    const removeQueries = selectedQueryIds.map((id) =>
      fleetQueriesAPI.destroy(id)
    );

    try {
      await Promise.all(removeQueries).then(() => {
        dispatch(
          renderFlash("success", `Successfully removed ${queryOrQueries}.`)
        );
        setResetSelectedRows(true);
        refetchFleetQueries();
      });
      dispatch(
        renderFlash("success", `Successfully removed ${queryOrQueries}.`)
      );
    } catch (errorResponse) {
      dispatch(
        renderFlash(
          "error",
          `There was an error removing your ${queryOrQueries}. Please try again later.`
        )
      );
    } finally {
      toggleRemoveQueryModal();
    }
  }, [dispatch, refetchFleetQueries, selectedQueryIds, toggleRemoveQueryModal]);

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

  const isTableDataLoading = isLoadingFleetQueries || queriesList === null;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
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
            <QueriesListWrapper
              queriesList={queriesList}
              isLoading={isTableDataLoading}
              onCreateQueryClick={onCreateQueryClick}
              onRemoveQueryClick={onRemoveQueryClick}
              searchable={!!queriesList}
              customControl={renderPlatformDropdown}
              selectedDropdownFilter={selectedDropdownFilter}
              isOnlyObserver={isOnlyObserver}
            />
          )}
        </div>
        {showRemoveQueryModal && (
          <RemoveQueryModal
            onCancel={toggleRemoveQueryModal}
            onSubmit={onRemoveQuerySubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManageQueriesPage;
