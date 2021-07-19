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

    this.setState({ querySearchText: queryData });
  };

  render() {
    const {
      onTableQueryChange,
      onRemoveScheduledQueries,
      isLoadingScheduledQueries,
      scheduledQueries,
    } = this;
    const { querySearchText } = this.state;

    // apply the filter to the props and render the filtered list

    // use querySearchText in state to apply the filter for the this.props.scheduledQueries;

    console.log(
      "scheduledlistwrapper this.props.scheduledqueries",
      scheduledQueries
    );

    const filtered = simpleSearch(querySearchText, scheduledQueries);

    const tableHeaders = generateTableHeaders();

    return (
      <div className={`${baseClass} body-wrap`}>
        <div>
          <h1>Queries</h1>
        </div>
        <TableContainer
          columns={tableHeaders}
          data={generateDataSet(filtered)}
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
