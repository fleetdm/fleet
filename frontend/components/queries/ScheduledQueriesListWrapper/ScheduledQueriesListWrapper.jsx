import React, { Component } from "react";
import PropTypes from "prop-types";

import simpleSearch from "utilities/simple_search";
import TableContainer from "components/TableContainer";
import helpers from "components/queries/ScheduledQueriesListWrapper/helpers";
import queryInterface from "interfaces/query";
import EmptyPack from "./EmptyPack";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PackQueriesTable/PackQueriesTableConfig";

const baseClass = "scheduled-queries-list-wrapper";
class ScheduledQueriesListWrapper extends Component {
  static propTypes = {
    onRemoveScheduledQueries: PropTypes.func,
    onScheduledQueryFormSubmit: PropTypes.func,
    onDblClickScheduledQuery: PropTypes.func,
    onSelectScheduledQuery: PropTypes.func,
    scheduledQueries: PropTypes.arrayOf(queryInterface),
    packId: PropTypes.number,
    isLoadingScheduledQueries: PropTypes.bool,
  };

  constructor(props) {
    super(props);

    this.state = {
      querySearchText: "",
      checkedScheduledQueryIDs: [],
    };
  }

  onRemoveScheduledQueries = (selectedItemIds) => {
    const {
      onRemoveScheduledQueries: handleRemoveScheduledQueries,
    } = this.props;

    return handleRemoveScheduledQueries(selectedItemIds);
  };

  // 7/15 ADDED for New table
  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  onTableQueryChange = (queryData) => {
    const { selectedFilter, dispatch, packId, scheduledQueries } = this.props;
    const {
      pageIndex,
      pageSize,
      searchQuery,
      sortHeader,
      sortDirection,
    } = queryData;
    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }

    if (!searchQuery) {
      this.setState({ querySearchText: "" });
      return;
    }

    this.setState({ querySearchText: searchQuery });
  };

  getQueries = () => {
    const { scheduledQueries } = this.props;
    const { querySearchText } = this.state;

    return simpleSearch(querySearchText, scheduledQueries);
  };

  render() {
    const {
      onTableQueryChange,
      onRemoveScheduledQueries,
      isLoadingScheduledQueries,
      scheduledQueries,
      getQueries,
    } = this;
    const { querySearchText } = this.state;

    const tableHeaders = generateTableHeaders();

    const tableData = generateDataSet(getQueries());

    return (
      <div className={`${baseClass} body-wrap`}>
        <div>
          <h1>Queries</h1>
        </div>
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={isLoadingScheduledQueries}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search queries"}
          onQueryChange={onTableQueryChange}
          onSelectActionClick={onRemoveScheduledQueries}
          resultsTitle={"queries"}
          emptyComponent={EmptyPack}
          showMarkAllPages={false}
          selectActionButtonText={"Remove"}
          searchable
          wideSearch
          disablePagination
        />
      </div>
    );
  }
}

export default ScheduledQueriesListWrapper;
