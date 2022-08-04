/* eslint-disable react/prop-types */
import React, { useCallback, useContext, useState } from "react";

import { AppContext } from "context/app";
import { IQuery } from "interfaces/query";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import generateTableHeaders from "./QueriesTableConfig";

import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "queries-table";
const noQueriesClass = "no-queries";
interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}
interface IQueriesTableProps {
  queriesList: IQueryTableData[] | null;
  isLoading: boolean;
  onDeleteQueryClick: (selectedTableQueryIds: number[]) => void;
  onCreateQueryClick: () => void;
  searchable: boolean;
  customControl?: () => JSX.Element;
  selectedDropdownFilter: string;
  isOnlyObserver?: boolean;
}

const QueriesTable = ({
  queriesList,
  isLoading,
  onDeleteQueryClick,
  onCreateQueryClick,
  searchable,
  customControl,
  selectedDropdownFilter,
  isOnlyObserver,
}: IQueriesTableProps): JSX.Element | null => {
  const { currentUser } = useContext(AppContext);
  const [searchString, setSearchString] = useState<string>("");

  const handleSearchChange = ({ searchQuery }: ITableQueryData) => {
    setSearchString(searchQuery);
  };

  const NoQueriesComponent = useCallback(() => {
    return (
      <div className={`${noQueriesClass}`}>
        <div className={`${noQueriesClass}__inner`}>
          <div className={`${noQueriesClass}__inner-text`}>
            {searchString ? (
              <>
                <h2>No queries match the current search criteria.</h2>
                <p>
                  Expecting to see queries? Try again in a few seconds as the
                  system catches up.
                </p>
              </>
            ) : (
              <>
                <h2>You don&apos;t have any queries.</h2>
                <p>
                  A query is a specific question you can ask about your devices.
                </p>
                {!isOnlyObserver && (
                  <>
                    <p>
                      Create a new query, or{" "}
                      <a
                        target="_blank"
                        rel="noreferrer noopener"
                        href="https://fleetdm.com/docs/using-fleet/standard-query-library"
                      >
                        import Fleet’s standard query library{" "}
                        <img
                          src={OpenNewTabIcon}
                          alt="open new tab"
                          id="new-tab-icon"
                        />
                      </a>
                    </p>
                    <Button
                      variant="brand"
                      className={`${baseClass}__create-button`}
                      onClick={onCreateQueryClick}
                    >
                      Create new query
                    </Button>
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    );
  }, [searchString, onCreateQueryClick]);

  const tableHeaders = currentUser && generateTableHeaders(currentUser);

  // Queries have not been created
  if (!isLoading && queriesList?.length === 0) {
    return (
      <div className={`${baseClass}`}>
        <NoQueriesComponent />
      </div>
    );
  }

  return tableHeaders && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={queriesList}
        isLoading={isLoading}
        defaultSortHeader={"updated_at"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={handleSearchChange}
        inputPlaceHolder="Search by name"
        searchable={searchable}
        disablePagination
        onPrimarySelectActionClick={onDeleteQueryClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={NoQueriesComponent}
        customControl={customControl}
        isClientSideFilter
        searchQueryColumn="name"
        selectedDropdownFilter={selectedDropdownFilter}
        isClientSidePagination
      />
    </div>
  ) : null;
};

export default QueriesTable;
