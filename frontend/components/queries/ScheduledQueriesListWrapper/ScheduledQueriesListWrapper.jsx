import React, { Component } from "react";
import PropTypes from "prop-types";
import { pull } from "lodash";

import Button from "components/buttons/Button";
import helpers from "components/queries/ScheduledQueriesListWrapper/helpers";
import InputField from "components/forms/fields/InputField";
import QueriesList from "components/queries/ScheduledQueriesList";
import queryInterface from "interfaces/query";

const baseClass = "scheduled-queries-list-wrapper";

class ScheduledQueriesListWrapper extends Component {
  static propTypes = {
    onRemoveScheduledQueries: PropTypes.func,
    onScheduledQueryFormSubmit: PropTypes.func,
    onDblClickScheduledQuery: PropTypes.func,
    onSelectScheduledQuery: PropTypes.func,
    scheduledQueries: PropTypes.arrayOf(queryInterface),
  };

  constructor(props) {
    super(props);

    this.state = {
      querySearchText: "",
      checkedScheduledQueryIDs: [],
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

  onCheckQuery = (shouldCheckQuery, scheduledQueryID) => {
    const { checkedScheduledQueryIDs } = this.state;
    const newCheckedScheduledQueryIDs = shouldCheckQuery
      ? checkedScheduledQueryIDs.concat(scheduledQueryID)
      : pull(checkedScheduledQueryIDs, scheduledQueryID);

    this.setState({ checkedScheduledQueryIDs: newCheckedScheduledQueryIDs });

    return false;
  };

  onUpdateQuerySearchText = (querySearchText) => {
    this.setState({ querySearchText });
  };

  getQueries = () => {
    const { scheduledQueries } = this.props;
    const { querySearchText } = this.state;

    return helpers.filterQueries(scheduledQueries, querySearchText);
  };

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
      onUpdateQuerySearchText,
      renderButton,
      renderQueryCount,
      renderQueriesList,
    } = this;
    const { querySearchText } = this.state;

    return (
      <div className={`${baseClass} body-wrap`}>
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
