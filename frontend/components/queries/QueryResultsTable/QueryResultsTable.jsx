import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { keys, omit } from "lodash";

import Button from "components/buttons/Button";
import campaignInterface from "interfaces/campaign";
import filterArrayByHash from "utilities/filter_array_by_hash";
import FleetIcon from "components/icons/FleetIcon";
import InputField from "components/forms/fields/InputField";
import QueryResultsRow from "components/queries/QueryResultsTable/QueryResultsRow";
import QueryProgressDetails from "components/queries/QueryProgressDetails";
import Spinner from "components/loaders/Spinner";

const baseClass = "query-results-table";

class QueryResultsTable extends Component {
  static propTypes = {
    campaign: campaignInterface.isRequired,
    onExportQueryResults: PropTypes.func,
    onExportErrorsResults: PropTypes.func,
    onToggleQueryFullScreen: PropTypes.func,
    isQueryFullScreen: PropTypes.bool,
    isQueryShrinking: PropTypes.bool,
    onRunQuery: PropTypes.func.isRequired,
    onStopQuery: PropTypes.func.isRequired,
    queryIsRunning: PropTypes.bool,
    queryTimerMilliseconds: PropTypes.number,
  };

  constructor(props) {
    super(props);

    this.state = {
      resultsFilter: {},
    };
  }

  onFilterAttribute = (attribute) => {
    return (value) => {
      const { resultsFilter } = this.state;

      this.setState({
        resultsFilter: {
          ...resultsFilter,
          [attribute]: value,
        },
      });

      return false;
    };
  };

  onSetActiveColumn = (activeColumn) => {
    return () => {
      this.setState({ activeColumn });
    };
  };

  renderTableHeaderColumn = (column, index) => {
    const filterable = column === "hostname" ? "host_hostname" : column;
    const { activeColumn, resultsFilter } = this.state;
    const { onFilterAttribute, onSetActiveColumn } = this;
    const filterIconClassName = classnames(`${baseClass}__filter-icon`, {
      [`${baseClass}__filter-icon--is-active`]: activeColumn === column,
    });

    return (
      <th key={`query-results-table-header-${index}`}>
        <span>
          <FleetIcon className={filterIconClassName} name="filter" />
          {column}
        </span>
        <InputField
          name={column}
          onChange={onFilterAttribute(filterable)}
          onFocus={onSetActiveColumn(column)}
          value={resultsFilter[filterable]}
        />
      </th>
    );
  };

  renderTableHeaderRow = (rows) => {
    const { renderTableHeaderColumn } = this;

    const queryAttrs = omit(rows[0], ["host_hostname"]);
    const queryResultColumns = keys(queryAttrs);

    return (
      <tr>
        {renderTableHeaderColumn("hostname", -1)}
        {queryResultColumns.map((column, i) => {
          return renderTableHeaderColumn(column, i);
        })}
      </tr>
    );
  };

  renderTableRows = (rows) => {
    const { resultsFilter } = this.state;
    const filteredRows = filterArrayByHash(rows, resultsFilter);

    return filteredRows.map((row) => {
      return <QueryResultsRow queryResult={row} />;
    });
  };

  renderTable = () => {
    const { renderTableHeaderRow, renderTableRows } = this;

    const { queryIsRunning, campaign } = this.props;
    const { query_results: queryResults } = campaign;
    const loading = queryIsRunning && (!queryResults || !queryResults.length);

    if (loading) {
      return <Spinner />;
    }

    return (
      <table className={`${baseClass}__table`}>
        <thead>{renderTableHeaderRow(queryResults)}</thead>
        <tbody>{renderTableRows(queryResults)}</tbody>
      </table>
    );
  };

  renderErrorsTable = () => {
    const { renderTableHeaderRow, renderTableRows } = this;

    const { queryIsRunning, campaign } = this.props;
    const { errors } = campaign;

    const loading = queryIsRunning && (!errors || !errors.length);

    if (loading) {
      return <Spinner />;
    }

    return (
      <table className={`${baseClass}__table`}>
        <thead>{renderTableHeaderRow(errors)}</thead>
        <tbody>{renderTableRows(errors)}</tbody>
      </table>
    );
  };

  render() {
    const {
      campaign,
      onExportQueryResults,
      onExportErrorsResults,
      isQueryFullScreen,
      isQueryShrinking,
      onToggleQueryFullScreen,
      onRunQuery,
      onStopQuery,
      queryIsRunning,
      queryTimerMilliseconds,
    } = this.props;

    const { renderTable, renderErrorsTable } = this;

    const {
      hosts_count: hostsCount,
      query_results: queryResults,
      errors,
    } = campaign;
    const hasNoResults =
      !queryIsRunning &&
      (!hostsCount.successful || !queryResults || !queryResults.length);
    const hasErrors = !queryIsRunning && errors;

    const resultsTableWrapClass = classnames(baseClass, {
      [`${baseClass}--full-screen`]: isQueryFullScreen,
      [`${baseClass}--shrinking`]: isQueryShrinking,
      [`${baseClass}__no-results`]: hasNoResults,
    });

    const toggleFullScreenBtnClass = classnames(
      `${baseClass}__fullscreen-btn`,
      {
        [`${baseClass}__fullscreen-btn--active`]: isQueryFullScreen,
      }
    );

    return (
      <div className={resultsTableWrapClass}>
        <header className={`${baseClass}__button-wrap`}>
          {isQueryFullScreen && (
            <QueryProgressDetails
              campaign={campaign}
              onRunQuery={onRunQuery}
              onStopQuery={onStopQuery}
              queryIsRunning={queryIsRunning}
              className={`${baseClass}__full-screen`}
              queryTimerMilliseconds={queryTimerMilliseconds}
            />
          )}
          <Button
            className={toggleFullScreenBtnClass}
            onClick={onToggleQueryFullScreen}
            variant="grey"
          >
            <FleetIcon name={isQueryFullScreen ? "windowed" : "fullscreen"} />
          </Button>
          {!hasNoResults && !queryIsRunning && (
            <Button
              className={`${baseClass}__export-btn`}
              onClick={onExportQueryResults}
              variant="inverse"
            >
              Export results
            </Button>
          )}
        </header>
        <span className={`${baseClass}__table-title`}>Results</span>
        <div className={`${baseClass}__results-table-wrapper`}>
          {hasNoResults && !hasErrors && (
            <span className="no-results-message">No results found.</span>
          )}
          {hasNoResults && hasErrors && (
            <span className="no-results-message">
              No results found. Check the table below for errors.
            </span>
          )}
          {!hasNoResults && renderTable()}
        </div>
        {hasErrors && (
          <div className={`${baseClass}__error-table-container`}>
            <header className={`${baseClass}__button-wrap`}>
              <Button
                className={`${baseClass}__export-btn`}
                onClick={onExportErrorsResults}
                variant="inverse"
              >
                Export errors
              </Button>
            </header>
            <span className={`${baseClass}__table-title`}>Errors</span>
            <div className={`${baseClass}__error-table-wrapper`}>
              {renderErrorsTable()}
            </div>
          </div>
        )}
      </div>
    );
  }
}

export default QueryResultsTable;
