import React, { useCallback, useEffect, useState } from "react";
import { useSelector } from "react-redux";

import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";
import permissionUtils from "utilities/permissions";
import TableContainer from "components/TableContainer";
import generateTableHeaders from "./QueriesTableConfig";

const baseClass = "queries-list-wrapper";
const noQueriesClass = "no-queries";

interface IQueriesListWrapperProps {
  onRemoveQueryClick: any;
  queriesList: IQuery[];
}

interface IRootState {
  auth: {
    user: IUser;
  };
  entities: {
    queries: {
      loading: boolean;
      data: IQuery[];
    };
  };
}

const QueriesListWrapper = (
  listProps: IQueriesListWrapperProps
): JSX.Element | null => {
  const { onRemoveQueryClick, queriesList } = listProps;

  const loadingQueries = useSelector(
    (state: IRootState) => state.entities.queries.loading
  );
  const [isLoading, setIsLoading] = useState<boolean>(true);
  useEffect(() => {
    setIsLoading(loadingQueries);
  }, [loadingQueries]);

  const currentUser = useSelector((state: IRootState) => state.auth.user);
  const isOnlyObserver = permissionUtils.isOnlyObserver(currentUser);

  const [filteredQueries, setFilteredQueries] = useState<IQuery[]>(queriesList);
  const [searchString, setSearchString] = useState<string>("");

  useEffect(() => {
    setFilteredQueries(
      !searchString
        ? queriesList
        : queriesList.filter((query) => {
            return query.name
              .toLowerCase()
              .includes(searchString.toLowerCase());
          })
    );
  }, [queriesList, searchString, setFilteredQueries]);

  const onQueryChange = useCallback(
    (queryData) => {
      const { searchQuery } = queryData;
      setSearchString(searchQuery);
    },
    [setSearchString]
  );

  const NoQueriesComponent = useCallback(() => {
    return (
      <div className={`${noQueriesClass}`}>
        <div className={`${noQueriesClass}__inner`}>
          <div className={`${noQueriesClass}__inner-text`}>
            {!searchString ? (
              <h2>You don&apos;t have any queries.</h2>
            ) : (
              <h2>No queries match your search.</h2>
            )}
            <p>
              Create a new query, or{" "}
              <a href="https://github.com/fleetdm/fleet/tree/main/docs/1-Using-Fleet/standard-query-library">
                go to GitHub
              </a>{" "}
              to import Fleet’s standard query library.
            </p>
          </div>
        </div>
      </div>
    );
  }, [searchString]);

  const tableHeaders = generateTableHeaders(isOnlyObserver);

  return !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={filteredQueries}
        isLoading={isLoading}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable
        disablePagination
        onPrimarySelectActionClick={onRemoveQueryClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={NoQueriesComponent}
      />
    </div>
  ) : null;
};

export default QueriesListWrapper;
