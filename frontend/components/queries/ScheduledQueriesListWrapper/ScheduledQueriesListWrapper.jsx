import React, { Component } from "react";
import PropTypes from "prop-types";
import { pull } from "lodash";

import simpleSearch from "utilities/simple_search";
import TableContainer from "components/TableContainer";
import Button from "components/buttons/Button";
import helpers from "components/queries/ScheduledQueriesListWrapper/helpers";
import InputField from "components/forms/fields/InputField";
import QueriesList from "components/queries/ScheduledQueriesList";
import queryInterface from "interfaces/query";
import scheduledQueryActions from "redux/nodes/entities/scheduled_queries/actions";
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
  };

  constructor(props) {
    super(props);

    this.state = {
      querySearchText: "",
      checkedScheduledQueryIDs: [],
      queryState: props.scheduledQueries,
    };
  }

  onRemoveScheduledQueries = (evt) => {
    evt.preventDefault();

    const {
      onRemoveScheduledQueries: handleRemoveScheduledQueries,
    } = this.props;
    const { checkedScheduledQueryIDs } = this.state;

    this.setState({ checkedScheduledQueryIDs: [] });

    return handleRemoveScheduledQueries(checkedScheduledQueryIDs);
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
      this.setState({ queryState: scheduledQueries });
      return;
    }

    this.setState({ queryState: simpleSearch(searchQuery, scheduledQueries) });
  };

  // Old table
  onCheckAllQueries = (shouldCheckAll) => {
    if (shouldCheckAll) {
      const allScheduledQueries = this.getQueries();
      const checkedScheduledQueryIDs = allScheduledQueries.map((sq) => sq.id);

      this.setState({ checkedScheduledQueryIDs });

      return false;
    }

    this.setState({ checkedScheduledQueryIDs: [] });

    return false;
  };

  // old table
  onCheckQuery = (shouldCheckQuery, scheduledQueryID) => {
    const { checkedScheduledQueryIDs } = this.state;
    const newCheckedScheduledQueryIDs = shouldCheckQuery
      ? checkedScheduledQueryIDs.concat(scheduledQueryID)
      : pull(checkedScheduledQueryIDs, scheduledQueryID);

    this.setState({ checkedScheduledQueryIDs: newCheckedScheduledQueryIDs });

    return false;
  };

  // old table
  onUpdateQuerySearchText = (querySearchText) => {
    this.setState({ querySearchText });
  };

  // old table
  getQueries = () => {
    const { scheduledQueries } = this.props;
    const { querySearchText } = this.state;

    return helpers.filterQueries(scheduledQueries, querySearchText);
  };

  // old table
  renderButton = () => {
    const { onRemoveScheduledQueries } = this;
    const { checkedScheduledQueryIDs } = this.state;

    const scheduledQueryCount = checkedScheduledQueryIDs.length;

    if (scheduledQueryCount) {
      const queryText = scheduledQueryCount === 1 ? "query" : "queries";

      return (
        <Button
          className={`${baseClass}__query-btn`}
          onClick={onRemoveScheduledQueries}
          variant="alert"
        >
          Remove {queryText}
        </Button>
      );
    }

    return false;
  };

  // old table
  renderQueryCount = () => {
    const { scheduledQueries } = this.props;
    const queryCount = scheduledQueries.length;
    const queryText = queryCount === 1 ? " 1 query" : `${queryCount} queries`;

    return (
      <div>
        <h1>Queries</h1>
        <p className={`${baseClass}__query-count`}>{queryText}</p>
      </div>
    );
  };

  // this is the old table
  renderQueriesList = () => {
    const {
      getQueries,
      onHidePackForm,
      onCheckAllQueries,
      onCheckQuery,
    } = this;
    const {
      onScheduledQueryFormSubmit,
      onSelectScheduledQuery,
      onDblClickScheduledQuery,
      scheduledQueries,
    } = this.props;
    const { checkedScheduledQueryIDs } = this.state;

    return (
      <div className={`${baseClass}__queries-list-wrapper`}>
        <QueriesList
          onHidePackForm={onHidePackForm}
          onScheduledQueryFormSubmit={onScheduledQueryFormSubmit}
          onCheckAllQueries={onCheckAllQueries}
          onCheckQuery={onCheckQuery}
          onSelectQuery={onSelectScheduledQuery}
          onDblClickQuery={onDblClickScheduledQuery}
          scheduledQueries={getQueries()}
          checkedScheduledQueryIDs={checkedScheduledQueryIDs}
          isScheduledQueriesAvailable={!!scheduledQueries.length}
        />
      </div>
    );
  };

  render() {
    const {
      onTableQueryChange,
      onUpdateQuerySearchText,
      renderButton,
      renderQueryCount,
      renderQueriesList,
      getQueries,
    } = this;
    const { querySearchText, queryState } = this.state;

    // hardcoded to false right now 7/15
    const loadingTableData = false;

    const tableHeaders = generateTableHeaders();

    return (
      <div className={`${baseClass} body-wrap`}>
        <TableContainer
          columns={tableHeaders}
          data={generateDataSet(queryState)}
          isLoading={loadingTableData}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search queries"}
          onQueryChange={onTableQueryChange}
          resultsTitle={"queries"}
          emptyComponent={EmptyPack}
          showMarkAllPages={false}
          selectActionButtonText={"Remove"}
          searchable
          wideSearch
          disablePagination
        />

        {renderQueryCount()}
        <div className={`${baseClass}__query-list-action`}>
          <InputField
            inputWrapperClass={`${baseClass}__search-queries-input`}
            name="search-queries"
            onChange={onUpdateQuerySearchText}
            placeholder="Search Queries"
            value={querySearchText}
          />
        </div>
        {renderQueriesList()}
        {renderButton()}
      </div>
    );
  }
}

export default ScheduledQueriesListWrapper;
