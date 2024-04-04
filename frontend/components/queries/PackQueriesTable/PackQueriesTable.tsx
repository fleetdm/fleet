import React, { useState } from "react";

import simpleSearch from "utilities/simple_search";
import { IScheduledQuery } from "interfaces/scheduled_query";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import Icon from "components/Icon/Icon";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PackQueriesTable/PackQueriesTableConfig";

const baseClass = "pack-queries-table";

interface IPackQueriesTableProps {
  onAddPackQuery: () => void;
  onEditPackQuery: (selectedQuery: IScheduledQuery) => void;
  onRemovePackQueries: (selectedTableQueryIds: number[]) => void;
  scheduledQueries: IScheduledQuery[] | undefined;
  isLoadingPackQueries: boolean;
}

const PackQueriesTable = ({
  onAddPackQuery,
  onEditPackQuery,
  onRemovePackQueries,
  scheduledQueries,
  isLoadingPackQueries,
}: IPackQueriesTableProps): JSX.Element => {
  const [querySearchText, setQuerySearchText] = useState("");

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onTableQueryChange = (queryData: ITableQueryData) => {
    const { searchQuery, sortHeader, sortDirection } = queryData;
    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }

    if (!searchQuery) {
      setQuerySearchText("");
      return;
    }

    setQuerySearchText(searchQuery);
  };

  const getQueries = () => {
    return simpleSearch(querySearchText, scheduledQueries);
  };

  const onActionSelection = (
    action: string,
    selectedQuery: IScheduledQuery
  ) => {
    switch (action) {
      case "edit":
        onEditPackQuery(selectedQuery);
        break;
      case "remove":
        onRemovePackQueries([selectedQuery.id]);
        break;
      default:
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const tableData = generateDataSet(getQueries());

  return (
    <div className={`${baseClass}`}>
      {scheduledQueries?.length ? (
        <TableContainer
          columnConfigs={tableHeaders}
          data={tableData}
          isLoading={isLoadingPackQueries}
          defaultSortHeader="name"
          defaultSortDirection="asc"
          inputPlaceHolder="Search queries"
          onQueryChange={onTableQueryChange}
          resultsTitle="queries"
          emptyComponent={() =>
            EmptyTable({
              header: "No queries match your search criteria",
              info: "Try a different search.",
            })
          }
          showMarkAllPages={false}
          actionButton={{
            name: "add query",
            buttonText: "Add query",
            iconSvg: "plus",
            variant: "text-icon",
            onActionButtonClick: onAddPackQuery,
          }}
          primarySelectAction={{
            name: "remove query",
            buttonText: "Remove",
            iconSvg: "close",
            variant: "text-icon",
            onActionButtonClick: onRemovePackQueries,
          }}
          searchable
          disablePagination
          isAllPagesSelected={false}
        />
      ) : (
        <div className={`${baseClass}__no-queries`}>
          <p>Your pack has no queries.</p>
          <Button
            onClick={onAddPackQuery}
            variant="text-icon"
            className={`${baseClass}__no-queries-action-button`}
          >
            <>
              Add query
              <Icon name="plus" />
            </>
          </Button>
        </div>
      )}
    </div>
  );
};

export default PackQueriesTable;
