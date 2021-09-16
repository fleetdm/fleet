import React, { useCallback, useEffect, useState } from "react";
import { useSelector } from "react-redux";

import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";
import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./QueriesTableConfig";

const baseClass = "queries-list-wrapper";
const noQueriesClass = "no-queries";

interface IQueriesListWrapperProps {
  onRemoveQueryClick: any;
  onCreateQueryClick: any;
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
  const { onRemoveQueryClick, onCreateQueryClick, queriesList } = listProps;

  const loadingQueries = useSelector(
    (state: IRootState) => state.entities.queries.loading
  );
  const [isLoading, setIsLoading] = useState<boolean>(true);
  useEffect(() => {
    setIsLoading(loadingQueries);
  }, [loadingQueries]);

  const currentUser = useSelector((state: IRootState) => state.auth.user);

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
              <>
                <h2>You don&apos;t have any queries.</h2>
                <p>
                  A query is a specific question you can ask about your devices.
                </p>
                <p>
                  Create a new query, or go to GitHub to{" "}
                  <a href="https://github.com/fleetdm/fleet/tree/main/docs/01-Using-Fleet/standard-query-library#importing-the-queries-in-fleet">
                    import Fleet’s standard query library
                  </a>
                  .
                </p>
                <Button
                  variant="brand"
                  className={`${baseClass}__create-button`}
                  onClick={onCreateQueryClick}
                >
                  Create new query
                </Button>
              </>
            ) : (
              <>
                <h2>No queries match the current search criteria.</h2>
                <p>
                  Expecting to see queries? Try again in a few seconds as the
                  system catches up.
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    );
  }, [searchString, onCreateQueryClick]);

  const tableHeaders = generateTableHeaders(isOnlyObserver);
  const dataSet = generateDataSet(filteredQueries);

  return !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={dataSet}
        isLoading={isLoading}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable={queriesList.length > 0}
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
