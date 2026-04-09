import React, { useState } from "react";

import simpleSearch from "utilities/simple_search";
import { IScheduledQuery } from "interfaces/scheduled_query";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
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
    const { searchQuery } = queryData;

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
          emptyComponent={() => (
            <EmptyState
              header="No queries match your search criteria"
              info="Try a different search."
            />
          )}
          showMarkAllPages={false}
          actionButton={{
            name: "add query",
            buttonText: "Add query",
            iconSvg: "plus",
            iconColor: "core-fleet-green",
            variant: "brand-inverse-icon",
            onClick: onAddPackQuery,
          }}
          primarySelectAction={{
            name: "remove query",
            buttonText: "Remove",
            iconSvg: "close",
            variant: "inverse",
            onClick: onRemovePackQueries,
          }}
          searchable
          disablePagination
          hideFooter
          isAllPagesSelected={false}
        />
      ) : (
        <EmptyState
          header="Your pack has no reports"
          primaryButton={
            <Button
              onClick={onAddPackQuery}
              variant="brand-inverse-icon"
              iconStroke
            >
              <>
                Add report
                <Icon name="plus" color="core-fleet-green" />
              </>
            </Button>
          }
        />
      )}
    </div>
  );
};

export default PackQueriesTable;
